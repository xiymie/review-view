package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"review-view/internal/config"
	"review-view/internal/handler"
	"review-view/internal/model"
	"review-view/internal/notify"
	"review-view/internal/review"
	"review-view/internal/service"
	"review-view/internal/store"
	gormstore "review-view/internal/store/gorm"
)

type Server struct {
	engine    *gin.Engine
	addr      string
	scheduler *service.Scheduler
	cache     *service.TaskCache
}

func NewRouterWithDependencies(handlers *handler.Handlers) *gin.Engine {
	return handler.NewRouter(handlers)
}

func NewServer(cfg config.Config) (*Server, error) {
	db, err := gormstore.Open(cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	stores := gormstore.New(db)
	settingsService := service.NewSettingsService(stores.GlobalConfigs)
	settings, err := settingsService.Get()
	if err != nil {
		return nil, err
	}

	if err := bootstrapAdmin(stores.Users); err != nil {
		return nil, err
	}

	repoManager := review.NewRepositoryManager(settings.RepoBaseDir, nil)
	taskService := service.NewTaskService(stores.Projects, stores.ModelConfigs, stores.Tasks, repoManager, stores.Credentials, settingsService)
	logBuffer := service.NewTaskCache(stores.Tasks)
	sensitiveWordService := service.NewSensitiveWordService(stores.SensitiveWords)
	scheduler := service.NewScheduler(
		stores.Projects,
		stores.ModelConfigs,
		stores.Tasks,
		stores.GlobalConfigs,
		repoManager,
		stores.Credentials,
		nil,
		logBuffer,
		int64(settings.MaxConcurrentTasks),
		5,
		sensitiveWordService,
	)
	scheduler.SetTaskService(taskService)

	// 构建推送 notifier：通过闭包每次发送时动态读取 SMTP 配置，支持运行时修改
	dispatcher := notify.NewDispatcher(
		notify.NewEmailNotifier(func() notify.SMTPConfig {
			host, port, user, pass, from, fromName, tls := settingsService.GetSMTPConfig()
			return notify.ParseSMTPConfig(host, port, user, pass, from, fromName, tls)
		}),
		notify.NewWecomNotifier(),
	)
	scheduler.SetNotifier(dispatcher, stores.Users)

	projectService := service.NewProjectService(stores.Projects, stores.ModelConfigs, stores.Tasks, repoManager, stores.Credentials)
	modelService := service.NewModelConfigService(stores.ModelConfigs)
	dashboardService := service.NewDashboardService(stores.Projects, stores.Tasks)
	credentialService := service.NewRepoCredentialService(stores.Credentials, stores.Projects)
	taskHandler := handler.NewTaskHandler(stores.Tasks, stores.Projects, taskService, scheduler, logBuffer, stores.Users)

	engine := NewRouterWithDependencies(&handler.Handlers{
		Dashboard:      handler.NewDashboardHandler(dashboardService, stores.Users),
		Projects:       handler.NewProjectHandler(projectService, modelService, taskService, stores.Tasks, credentialService, stores.Users),
		Models:         handler.NewModelHandler(modelService),
		Settings:       handler.NewSettingsHandler(settingsService),
		Tasks:          taskHandler,
		Webhook:        handler.NewWebhookHandler(taskService),
		Credentials:    handler.NewCredentialHandler(credentialService, stores.Users),
		Auth:           handler.NewAuthHandler(stores.Users),
		SensitiveWords: handler.NewSensitiveWordHandler(sensitiveWordService),
		Users:          handler.NewUserHandler(stores.Users, settingsService),
	})

	// 清理上次运行残留的 running 任务
	_, _ = stores.Tasks.RecoverRunning()

	// 清理上次运行残留的 initializing 项目
	_, _ = stores.Projects.RecoverInitializing()

	return &Server{
		engine:    engine,
		addr:      cfg.Addr,
		scheduler: scheduler,
		cache:     logBuffer,
	}, nil
}

func (s *Server) Handler() *gin.Engine {
	return s.engine
}

func (s *Server) Run() error {
	ctx := context.Background()
	go s.cache.StartFlushLoop(ctx)
	go s.scheduler.Loop(ctx)
	return s.engine.Run(s.addr)
}

func bootstrapAdmin(users store.UserStore) error {
	existing, err := users.GetByUsername("admin")
	if err == nil {
		// admin exists — ensure it has super_admin role
		if existing.Role != model.UserRoleSuperAdmin {
			existing.Role = model.UserRoleSuperAdmin
			return users.Update(existing)
		}
		return nil
	}
	// No admin user yet — create one
	count, err := users.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("Snto123!@#"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return users.Create(&model.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         model.UserRoleSuperAdmin,
	})
}
