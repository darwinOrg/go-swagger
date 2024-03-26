package swagger_test

import (
	"github.com/darwinOrg/go-common/page"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-swagger"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
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

	swagger.SyncSwaggerToApifox(&swagger.SyncToApifoxRequest{
		RequestApis: wrapper.GetRequestApis(),
		ProjectId:   "3450238",
		//AccessToken: "APS-d4KgT80K2Wu89UAUc6r94NchTH6SJeFM",
		AccessToken:         "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6NDIyNzIxLCJ0cyI6IjgwYjI4MmIzOTNkMGY2MmMiLCJpYXQiOjE2OTUyODkxOTI0NzF9.U5ly2UQ0rpTIO_zdh68_pGvw7PvkZlW3lwTbg-cWySU",
		ApiOverwriteMode:    swagger.ApiOverwriteModeIgnore,
		SchemaOverwriteMode: swagger.SchemaOverwriteModeIgnore,
		SyncApiFolder:       false,
		ImportBasePath:      false,
		ApiFolderPath:       "测试1/测试2",
	})
}
