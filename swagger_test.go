package swagger_test

import (
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/page"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-swagger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestExport(t *testing.T) {
	engine := wrapper.DefaultEngine()

	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[*result.Void]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "/get",
		NonLogin:     true,
		BizHandler: func(_ *gin.Context, ctx *dgctx.DgContext, request *wrapper.MapRequest) *result.Result[*result.Void] {
			return result.SimpleSuccess()
		},
	})

	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*page.PageList[*UserRequest]]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "post",
		NonLogin:     true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *UserRequest) *result.Result[*page.PageList[*UserRequest]] {
			return result.Success[*page.PageList[*UserRequest]](nil)
		},
	})

	swagger.ExportSwaggerFile(&swagger.ExportSwaggerRequest{
		ServiceName: "test-service",
		//Title:       "测试服务标题",
		//Description: "测试服务描述",
		//OutDir:      "openapi/v1",
		//Version:     "v0.0.1",
		RequestApis: wrapper.GetRequestApis(),
	})
}

type UserRequest struct {
	Name     string    `binding:"required" errMsg:"姓名错误:不能为空" remark:"名称"`
	Age      int       `binding:"required,gt=0,lt=100" remark:"年龄"`
	UserInfo *userInfo `binding:"required"`
}

type userInfo struct {
	Sex int `binding:"required,gt=0,lt=5" errMsg:"性别错误" remark:"性别"`
}