package agentcli

// Hook allows commands to run standardized pre/post execution steps.
type Hook interface {
	Preflight(*AppContext) error
	Postflight(*AppContext) error
}

// RunLifecycle executes preflight, run, and postflight in order.
func RunLifecycle(app *AppContext, hook Hook, run func(*AppContext) error) error {
	if app == nil {
		app = NewAppContext(nil)
	}
	if hook != nil {
		if err := hook.Preflight(app); err != nil {
			return err
		}
	}
	if run != nil {
		if err := run(app); err != nil {
			return err
		}
	}
	if hook != nil {
		if err := hook.Postflight(app); err != nil {
			return err
		}
	}
	return nil
}
