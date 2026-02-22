package app

type Hooks struct{}

func (Hooks) Preflight() error {
	return nil
}

func (Hooks) Postflight() error {
	return nil
}
