package fakes

type StatusClient struct {
	LeaderCall struct {
		CallCount int
		Returns   struct {
			Leader string
			Error  error
		}
	}
}

func (c *StatusClient) Leader() (string, error) {
	c.LeaderCall.CallCount++
	return c.LeaderCall.Returns.Leader, c.LeaderCall.Returns.Error
}
