package api

import (
	adminCtrl "forxi.cn/forxi-go/app/controller/admin"
	userCtrl "forxi.cn/forxi-go/app/controller/user"
	"forxi.cn/forxi-go/app/middleware"
	"forxi.cn/forxi-go/app/resource"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func SetupRoutes(router *gin.Engine) {
	cfg := resource.Cfg
	rateLimiter := middleware.NewIPRateLimiter(rate.Limit(cfg.RateLimit.QPS), cfg.RateLimit.Burst)

	userController := userCtrl.NewUserController()
	authController := userCtrl.NewAuthController()
	adminController := adminCtrl.NewAdminController()
	filePreviewController := userCtrl.NewFilePreviewController()
	uploadController := userCtrl.NewUploadController()

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.GinLogger())
	router.Use(middleware.RateLimitMiddleware(rateLimiter))

	apiGroup := router.Group("/api")
	{
		public := apiGroup.Group("")
		{
			public.POST("/users/register", userController.Register)
			public.POST("/users/send-code", userController.SendRegisterCode)
			public.POST("/upload", middleware.OptionalAuthMiddleware(cfg.JWT.Secret), uploadController.Upload)

			filereview := public.Group("/filereview")
			{
				filereview.GET("/online", filePreviewController.Online)
				filereview.POST("/local", filePreviewController.Local)
			}

			auth := public.Group("/auth")
			{
				auth.POST("/login", authController.Login)
				auth.POST("/refresh", authController.RefreshToken)
				auth.POST("/password/reset", authController.RequestPasswordReset)
				auth.POST("/password/reset/confirm", authController.ResetPassword)
			}

			oauth := public.Group("/oauth")
			{
				oauth.GET("/github/authorize", middleware.OptionalAuthMiddleware(cfg.JWT.Secret), authController.GetGitHubAuthURL)
				oauth.GET("/github/callback", authController.GitHubLogin)
				oauth.POST("/github/bind-email", authController.BindGitHubEmail)
			}
		}

		authRequired := apiGroup.Group("")
		authRequired.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			users := authRequired.Group("/users")
			{
				users.GET("/profile", userController.GetProfile)
				users.PUT("/profile", userController.UpdateProfile)
				users.PUT("/password", userController.ChangePassword)
			}

			auth := authRequired.Group("/auth")
			{
				auth.POST("/logout", authController.Logout)
				auth.GET("/login-logs", authController.GetLoginLogs)
			}

			oauth := authRequired.Group("/oauth")
			{
				oauth.POST("/unbind", authController.UnbindOAuthAccount)
				oauth.GET("/accounts", authController.GetOAuthAccounts)
			}
		}
	}

	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		adminGroup.Use(middleware.RequireRole("admin", "super_admin"))
		{
			adminGroup.GET("/users", adminController.ListUsers)
			adminGroup.PUT("/users/:id/status", adminController.UpdateUserStatus)

			superAdmin := adminGroup.Group("")
			superAdmin.Use(middleware.RequireRole("super_admin"))
			{
				superAdmin.PUT("/users/:id/role", adminController.UpdateUserRole)
			}
		}
	}
}
