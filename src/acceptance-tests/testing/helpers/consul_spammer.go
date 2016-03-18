package helpers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
)

const (
	SUCCESSFUL_KEY_WRITE_THRESHOLD = 0.85
	MAX_SUCCESSIVE_RPC_ERROR_COUNT = 6
)

type counts struct {
	keyCount  int
	rpcErrors int
}

func SpamConsul(done chan struct{}, wg *sync.WaitGroup, consulClient consul.HTTPKV) chan map[string]string {
	keyValChan := make(chan map[string]string, 1)
	wg.Add(1)

	go func() {
		counts := counts{}
		keyVal := make(map[string]string)
		address := strings.TrimSuffix(consulClient.ConsulAddress, "/consul")
		address = strings.TrimPrefix(address, "http://")
		for {
			select {
			case <-done:
				keyValChan <- keyVal

				successRate := float32(len(keyVal)) / float32(counts.keyCount)

				if successRate < SUCCESSFUL_KEY_WRITE_THRESHOLD {
					keyVal["error"] = fmt.Sprintf("too many keys failed to write: %.2f failure rate", 1-successRate)
				}

				wg.Done()
				return
			case <-time.After(1 * time.Second):
				guid, err := NewGUID()
				if err != nil {
					keyVal["error"] = err.Error()
					continue
				}

				key := fmt.Sprintf("consul-key-%s", guid)
				value := fmt.Sprintf("consul-value-%s", guid)

				counts.keyCount++
				err = consulClient.Set(key, value)
				if err != nil {
					switch {
					case strings.Contains(err.Error(), "rpc error"):
						counts.rpcErrors++
					case strings.Contains(err.Error(), fmt.Sprintf("dial tcp %s: getsockopt: connection refused", address)):
						// failures to connect to the test consumer should not count as failed key writes
						// this typically happens when the test-consumer vm is rolled
						counts.keyCount--
					case strings.Contains(err.Error(), "http: proxy error:"):
					default:
						keyVal["error"] = err.Error()
					}

					if counts.rpcErrors > MAX_SUCCESSIVE_RPC_ERROR_COUNT {
						keyVal["error"] = "too many rpc errors"
					}
					continue
				}

				keyVal[key] = value
				counts.rpcErrors = 0
			}
		}
	}()

	return keyValChan
}
