package chaperon

import (
	"strings"

	"code.cloudfoundry.org/lager"
)

type statusClient interface {
	Leader() (string, error)
}

type BootstrapChecker struct {
	agentClient  agentClient
	statusClient statusClient
	logger       logger
}

func NewBootstrapChecker(logger logger, agentClient agentClient, statusClient statusClient) BootstrapChecker {
	return BootstrapChecker{
		agentClient:  agentClient,
		statusClient: statusClient,
		logger:       logger,
	}
}

func (b BootstrapChecker) StartInBootstrapMode() (startInBootstrapMode bool, err error) {
	startInBootstrapMode = true

	defer func() {
		b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode", lager.Data{"bootstrap": startInBootstrapMode})
	}()

	b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode.agent-client.members")
	members, err := b.agentClient.Members(false)
	if err != nil {
		startInBootstrapMode = false
		b.logger.Error("chaperon-bootstrap-checker.start-in-bootstrap-mode.agent-client.members.failed", err)
		return
	}

	for _, member := range members {
		if member.Tags["bootstrap"] == "1" {
			startInBootstrapMode = false
			b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode.bootstrap-node-exists", lager.Data{"bootstrap-node": member.Name})
			return
		}
	}

	b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode.status-client.leader")
	var leader string
	leader, err = b.statusClient.Leader()
	if err != nil {
		if strings.Contains(err.Error(), "No known Consul servers") {
			b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode.status-client.leader.no-known-consul-servers")
			return startInBootstrapMode, nil
		}

		startInBootstrapMode = false
		b.logger.Error("chaperon-bootstrap-checker.start-in-bootstrap-mode.status-client.leader.failed", err)
		return
	}

	if leader != "" {
		startInBootstrapMode = false
		b.logger.Info("chaperon-bootstrap-checker.start-in-bootstrap-mode.leader-exists", lager.Data{"leader": leader})
		return
	}

	return
}
