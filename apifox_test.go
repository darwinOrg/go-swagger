package swagger_test

import (
	"github.com/darwinOrg/go-common/page"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-swagger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"os"
	"testing"
)

func TestSyncToApifoxRequest(t *testing.T) {
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

	swagger.SyncRequestApisToApifox(&swagger.SyncToApifoxRequest{
		ProjectId:           "3450238",
		AccessToken:         os.Getenv("APIFOX_TOKEN"),
		ApiOverwriteMode:    swagger.ApiOverwriteModeIgnore,
		SchemaOverwriteMode: swagger.SchemaOverwriteModeIgnore,
		SyncApiFolder:       false,
		ImportBasePath:      false,
		ApiFolderPath:       "测试1/测试2",
	}, wrapper.GetRequestApis())
}

func TestSyncSwaggerJsonFileToApifox(t *testing.T) {
	swagger.SyncSwaggerJsonFileToApifox(&swagger.SyncToApifoxRequest{
		ProjectId:           "3450238",
		AccessToken:         os.Getenv("APIFOX_TOKEN"),
		ApiOverwriteMode:    swagger.ApiOverwriteModeIgnore,
		SchemaOverwriteMode: swagger.SchemaOverwriteModeIgnore,
		SyncApiFolder:       false,
		ImportBasePath:      false,
		ApiFolderPath:       "测试1/测试2",
	}, "test.swagger.json")
}
