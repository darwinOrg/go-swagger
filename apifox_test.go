package swagger_test

import (
	"os"
	"testing"

	"github.com/darwinOrg/go-common/page"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-swagger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
)

type TagVo struct {
	Id              int64  `json:"id" remark:"id"`
	Name            string `json:"name" remark:"名称"`
	CreatedAt       string `json:"createdAt,omitempty" remark:"创建时间"`
	CreatedUserName string `json:"createdUserName,omitempty" remark:"创建人名称"`
}

type TagTreeVo struct {
	*TagVo
	Children []*TagTreeVo `json:"children" remark:"子标签"`
}

func TestSyncToApifoxRequest(t *testing.T) {
	engine := gin.Default()

	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[*result.Void]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "/get",
	})

	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*page.PageList[*TagTreeVo]]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/test"),
		RelativePath: "post",
	})

	swagger.SyncRequestApisToApifox(&swagger.SyncToApifoxRequest{
		ProjectId:           os.Getenv("APIFOX_PROJECT_ID"),
		AccessToken:         os.Getenv("APIFOX_ACCESS_TOKEN"),
		ApiOverwriteMode:    swagger.ApiOverwriteModeMethodAndPath,
		SchemaOverwriteMode: swagger.SchemaOverwriteModeIgnore,
		SyncApiFolder:       false,
		ImportBasePath:      false,
		ApiFolderPath:       "测试1/测试2",
	}, wrapper.GetRequestApis())
}

func TestSyncSwaggerJsonFileToApifox(t *testing.T) {
	swagger.SyncSwaggerJsonFileToApifox(&swagger.SyncToApifoxRequest{
		ProjectId:           os.Getenv("APIFOX_PROJECT_ID"),
		AccessToken:         os.Getenv("APIFOX_ACCESS_TOKEN"),
		ApiOverwriteMode:    swagger.ApiOverwriteModeIgnore,
		SchemaOverwriteMode: swagger.SchemaOverwriteModeIgnore,
		SyncApiFolder:       false,
		ImportBasePath:      false,
		ApiFolderPath:       "测试1/测试2",
	}, "test.swagger.json")
}
