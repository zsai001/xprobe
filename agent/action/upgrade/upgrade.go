package upgrade

type UpgradeAction struct {
}

func (a *UpgradeAction) Execute(topic string, data interface{}) error {
	return nil
}
