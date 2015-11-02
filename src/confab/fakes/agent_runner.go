package fakes

type AgentRunner struct {
	RunCalls struct {
		CallCount int
		Returns   struct {
			Errors []error
		}
	}

	StopCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}
}

func (r *AgentRunner) Run() error {
	err := r.RunCalls.Returns.Errors[r.RunCalls.CallCount]
	r.RunCalls.CallCount++
	return err
}

func (r *AgentRunner) Stop() error {
	r.StopCall.CallCount++
	return r.StopCall.Returns.Error
}

type AgentClient struct {
	VerifyJoinedCalls struct {
		CallCount int
		Returns   struct {
			Errors []error
		}
	}
	VerifySyncedCalls struct {
		CallCount int
		Returns   struct {
			Errors []error
		}
	}

	IsLastNodeCall struct {
		Returns struct {
			IsLastNode bool
			Error      error
		}
	}

	SetKeysCall struct {
		Receives struct {
			Keys []string
		}
		Returns struct {
			Error error
		}
	}

	LeaveCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}
}

func (c *AgentClient) VerifyJoined() error {
	err := c.VerifyJoinedCalls.Returns.Errors[c.VerifyJoinedCalls.CallCount]
	c.VerifyJoinedCalls.CallCount++
	return err
}

func (c *AgentClient) VerifySynced() error {
	err := c.VerifySyncedCalls.Returns.Errors[c.VerifySyncedCalls.CallCount]
	c.VerifySyncedCalls.CallCount++
	return err
}

func (c *AgentClient) IsLastNode() (bool, error) {
	return c.IsLastNodeCall.Returns.IsLastNode, c.IsLastNodeCall.Returns.Error
}

func (c *AgentClient) SetKeys(keys []string) error {
	c.SetKeysCall.Receives.Keys = keys
	return c.SetKeysCall.Returns.Error
}

func (c *AgentClient) Leave() error {
	c.LeaveCall.CallCount++
	return c.LeaveCall.Returns.Error
}
