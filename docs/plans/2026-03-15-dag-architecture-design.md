# DAG Architecture Design: handler -> service -> operator -> dal

**Date:** 2026-03-15
**Status:** Approved
**Reference:** xj_ops DAG pattern

## Summary

Restructure agentcli-go into a DAG-layered architecture (cmd -> service -> operator -> dal) with Wire DI, and make generated projects (`agentcli new`) follow the same pattern by default.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Both framework + generated projects | Consistent mental model everywhere |
| DI framework | google/wire | Compile-time, same as xj_ops |
| Handler layer | cmd/ IS the handler (no separate handler/ pkg) | Library doesn't need standalone handler pkg; YAGNI |
| Root package | Model layer (shared contracts) | AppContext, Hook, CLIError stay importable; partial break only |
| DAL backends | Filesystem + exec + future external | Extensible without over-engineering now |
| Scaffold default | DAG is the new default | Replace current default; --minimal stays |

## Package Structure

### Framework (agentcli-go)

```
agentcli-go/
├── *.go                          ← ROOT = MODEL layer
│   ├── core_context.go           │  AppContext, IOStreams, AppMeta
│   ├── lifecycle.go              │  Hook interface (contract only)
│   └── errors.go                 │  CLIError, ExitCoder, exit codes
│
├── service/                      ← SERVICE layer: orchestration
│   ├── scaffold.go               │  ScaffoldNew, ScaffoldAddCommand
│   ├── doctor.go                 │  Doctor compliance checks
│   ├── lifecycle.go              │  RunLifecycle implementation
│   ├── container.go              │  Wire DI container
│   ├── wire.go                   │  Wire provider sets
│   └── wire_gen.go               │  (generated)
│
├── operator/                     ← OPERATOR layer: business logic
│   ├── interfaces.go             │  TemplateOperator, ComplianceOperator
│   ├── template_op.go            │  Template rendering, go mod inject
│   ├── compliance_op.go          │  File checks, marker validation
│   └── args_op.go                │  ParseArgs, RequireArg, GetArg, HasFlag
│
├── dal/                          ← DAL layer: I/O abstraction
│   ├── interfaces.go             │  FileSystem, Executor, BuildInfo
│   ├── filesystem.go             │  FileExists, EnsureDir, ReadFile, WriteFile
│   ├── exec.go                   │  RunCommand, RunOsascript, Which
│   └── logger.go                 │  InitLogger (zerolog setup)
│
├── cobrax/                       ← stays (adapter)
├── configx/                      ← stays (adapter)
├── cmd/agentcli/                 ← HANDLER layer (cmd = handler)
├── internal/                     ← stays (dogfood, harnessloop, migrate)
└── tools/harness/                ← stays
```

### Dependency DAG

```
cmd/agentcli  ──→  service  ──→  operator  ──→  dal
     │                │              │            │
     └────────────────┴──────────────┴────────────┘
                          │
                     root (model)
                 AppContext, Hook, CLIError
```

Rules:
- Root (model): imported by ALL layers — shared contracts only
- dal: imports root only — no knowledge of operator/service
- operator: imports root + dal — transforms data, applies rules
- service: imports root + operator + dal — orchestrates multi-step flows
- cmd (handler): imports service (and root for types) — CLI wiring

## Wire DI

### Container (service/container.go)

```go
type Container struct {
    // DAL
    FS   dal.FileSystem
    Exec dal.Executor

    // Operators
    TemplateOp   operator.TemplateOperator
    ComplianceOp operator.ComplianceOperator
    ArgsOp       operator.ArgsOperator

    // Services
    ScaffoldSvc  *ScaffoldService
    DoctorSvc    *DoctorService
    LifecycleSvc *LifecycleService
}

var globalContainer *Container

func Get() *Container {
    if globalContainer == nil {
        globalContainer = InitializeContainer()
    }
    return globalContainer
}
```

### Wire Providers (service/wire.go)

```go
var ProviderSet = wire.NewSet(
    // DAL
    dal.NewFileSystem,
    wire.Bind(new(dal.FileSystem), new(*dal.FileSystemImpl)),
    dal.NewExecutor,
    wire.Bind(new(dal.Executor), new(*dal.ExecutorImpl)),

    // Operators
    operator.NewTemplateOperator,
    wire.Bind(new(operator.TemplateOperator), new(*operator.TemplateOperatorImpl)),
    operator.NewComplianceOperator,
    wire.Bind(new(operator.ComplianceOperator), new(*operator.ComplianceOperatorImpl)),
    operator.NewArgsOperator,

    // Services
    NewScaffoldService,
    NewDoctorService,
    NewLifecycleService,

    // Container
    NewContainer,
)
```

## Generated Project Structure

`agentcli new my-tool` produces:

```
my-tool/
├── main.go
├── go.mod / go.sum
├── cmd/
│   └── root.go                      ← HANDLER: cobrax command registry
├── service/
│   ├── container.go                 ← Wire container + Get()
│   ├── wire.go                      ← Wire provider set
│   └── wire_gen.go                  ← (generated)
├── operator/
│   ├── interfaces.go                ← Operator contracts
│   └── example_op.go               ← Stub with TODO
├── dal/
│   ├── interfaces.go                ← DAL contracts
│   └── filesystem.go               ← Default filesystem impl
├── internal/
│   ├── app/
│   │   ├── bootstrap.go
│   │   └── lifecycle.go
│   ├── config/
│   │   ├── schema.go
│   │   └── load.go
│   └── tools/smokecheck/main.go
├── pkg/version/version.go
├── test/
│   ├── e2e/cli_test.go
│   └── smoke/version.schema.json
└── Taskfile.yml                     ← includes wire task
```

Data flow: `cmd/ → service.Get().XxxSvc.Method() → operator → dal`

## Function Migration Map

| Current location | Current export | New location | New identity |
|-----------------|---------------|-------------|-------------|
| root | AppContext, IOStreams, AppMeta | root (stays) | Model contract |
| root | Hook interface | root (stays) | Model contract |
| root | CLIError, ExitCoder, exit codes | root (stays) | Model contract |
| root | RunLifecycle | service/ | LifecycleService.Run() |
| root | ScaffoldNew, ScaffoldAddCommand | service/ | ScaffoldService methods |
| root | Doctor | service/ | DoctorService.Run() |
| root | ParseArgs, RequireArg, GetArg, HasFlag | operator/ | ArgsOperator methods |
| root | RunCommand, RunOsascript, Which, CheckDependency | dal/ | Executor interface + impl |
| root | FileExists, EnsureDir, GetBaseName | dal/ | FileSystem interface + impl |
| root | InitLogger | dal/ | Logger setup |

## Testing Strategy

| Layer | Test approach | Mocking |
|-------|--------------|---------|
| dal/ | Integration — real filesystem (temp dirs), real exec | None — IS the I/O boundary |
| operator/ | Unit — mock dal interfaces | dal.FileSystem mock, dal.Executor mock |
| service/ | Unit — mock operator interfaces | operator mocks |
| cmd/ (handler) | E2E — run binary, check stdout/exit codes | None — test through real stack |

## Error Handling

Errors wrap at each layer boundary with `fmt.Errorf("...: %w", err)`:
- dal: `return fmt.Errorf("read %s: %w", path, err)`
- operator: `return fmt.Errorf("render template: %w", err)`
- service: `return fmt.Errorf("scaffold project: %w", err)`
- cmd: maps to CLIError with exit code via existing ResolveExitCode

## Doctor Additions

New DAG compliance checks for generated projects:
- `service/wire.go` exists with ProviderSet
- `dal/interfaces.go` exists
- `operator/interfaces.go` exists
- No import violations (dal !-> operator/service, operator !-> service)
