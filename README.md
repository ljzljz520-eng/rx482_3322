# 医疗陪诊客户工作台

这是一个使用 Go 1.22.12 标准库实现的医疗陪诊客户维护服务。客服可以维护客户与患者关系、服务城市、回访时间和备注，并通过导入预览确认重复手机号后再提交数据；所有工作簿操作都会进入内存操作日志。

## 运行

在模块根目录执行：

```sh
export GOTOOLCHAIN=local
go run .
```

服务默认监听 `http://127.0.0.1:8080`，打开根路径即可使用 `web/` 工作台。接口包括：

可以通过 `PORT` 环境变量调整监听端口。

- `GET /api/customers`、`POST /api/customers`、`PUT /api/customers/{id}`
- `POST /api/customers/import/preview`
- `POST /api/customers/import`
- `GET /api/operation-logs`

业务链路测试命令：

```sh
GOTOOLCHAIN=local go test -count=1 ./...
```

项目固定使用内存仓储和确定性操作日志，不需要数据库、外部 API 或随机数据。编译时可使用 `CGO_ENABLED=0`。
