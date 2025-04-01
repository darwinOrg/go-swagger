package swagger_test

import (
	"github.com/darwinOrg/go-common/page"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-common/utils"
	"github.com/darwinOrg/go-swagger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestExposeGinSwagger(t *testing.T) {
	engine := gin.Default()

	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[*result.Void]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "/get",
	})

	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*page.PageList[*UserRequest]]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "post",
	})

	swagger.ExposeGinSwagger(engine)
	_ = engine.Run(":8080")
}

func TestExportSwaggerFile(t *testing.T) {
	engine := gin.Default()

	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[*result.Void]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "/get",
	})

	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*page.PageList[*UserRequest]]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "post",
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

func TestCreateSchemaForObject(t *testing.T) {
	schema := swagger.CreateSchemaForObject(&UserRequest{})
	t.Log(utils.MustConvertBeanToJsonString(schema))
}

type UserRequest struct {
	Name     string    `binding:"required" errMsg:"姓名错误:不能为空" title:"名称" remark:"名称"`
	Age      int       `binding:"required,gt=0,lt=100" title:"年龄" remark:"年龄"`
	UserInfo *userInfo `binding:"required"`
}

type userInfo struct {
	Sex int `binding:"required,gt=0,lt=5" errMsg:"性别错误" title:"性别" remark:"性别，0：男，1：女"`
}
