package service

import (
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

// Container is the Wire-managed dependency injection container for the service layer.
type Container struct {
	FS   dal.FileSystem
	Exec dal.Executor
	Log  dal.Logger

	TemplateOp   operator.TemplateOperator
	ComplianceOp operator.ComplianceOperator
	ArgsOp       operator.ArgsOperator

	ScaffoldSvc  *ScaffoldService
	DoctorSvc    *DoctorService
	LifecycleSvc *LifecycleService
}

// NewContainer creates a fully wired Container.
func NewContainer(
	fs dal.FileSystem,
	exec dal.Executor,
	lg dal.Logger,
	tpl operator.TemplateOperator,
	comp operator.ComplianceOperator,
	args operator.ArgsOperator,
	scaffold *ScaffoldService,
	doctor *DoctorService,
	lifecycle *LifecycleService,
) *Container {
	return &Container{
		FS: fs, Exec: exec, Log: lg,
		TemplateOp: tpl, ComplianceOp: comp, ArgsOp: args,
		ScaffoldSvc: scaffold, DoctorSvc: doctor, LifecycleSvc: lifecycle,
	}
}

var globalContainer *Container

// Get returns the global Container, initializing it lazily via Wire.
func Get() *Container {
	if globalContainer == nil {
		globalContainer = InitializeContainer()
	}
	return globalContainer
}

// Reset clears the global Container (useful for testing).
func Reset() {
	globalContainer = nil
}
