for /f "tokens=1-4 delims=/ " %%i in ("%date%") do (
     set dow=%%i
     set month=%%j
     set day=%%k
     set year=%%l
)
set datestr=%day%.%month%.%year%
cd web
swag init
cd ..
go build -ldflags "-X 'main.compileDate=%datestr%'" web/main.go web/log.go web/handlers.go web/middleware.go web/routers.go web/db.go

main