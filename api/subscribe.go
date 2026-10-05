package api

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"mime"
	"net/http"
	"trojan-panel/model/constant"
	"trojan-panel/model/dto"
	"trojan-panel/model/vo"
	"trojan-panel/service"
	"trojan-panel/util"
)

// ClashSubscribe 获取Clash订阅地址
func ClashSubscribe(c *gin.Context) {
	accountVo := service.GetCurrentAccount(c)
	password, err := service.SelectConnectPassword(&accountVo.Id, &accountVo.Username)
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(subscriptionPath(password, c.Query("target")), c)
}

// ClashSubscribeForSb 获取指定人的Clash订阅地址
func ClashSubscribeForSb(c *gin.Context) {
	var accountRequiredIdDto dto.RequiredIdDto
	_ = c.ShouldBindQuery(&accountRequiredIdDto)
	if err := validate.Struct(&accountRequiredIdDto); err != nil {
		vo.Fail(constant.ValidateFailed, c)
		return
	}
	password, err := service.SelectConnectPassword(accountRequiredIdDto.Id, nil)
	if err != nil {
		vo.Fail(err.Error(), c)
		return
	}
	vo.Success(subscriptionPath(password, c.Query("target")), c)
}

// Subscribe 订阅
func Subscribe(c *gin.Context) {
	token := c.Param("token")
	tokenDecode, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid subscription token")
		return
	}
	pass := string(tokenDecode)

	account, userInfo, clashConfigYaml, systemConfig, err := service.SubscribeClash(pass)
	if err != nil {
		c.String(http.StatusBadRequest, "Unable to load subscription")
		return
	}
	result, err := util.BuildClashProfile(clashConfigYaml, systemConfig.ClashRule, c.Query("target"))
	if err != nil {
		c.String(http.StatusUnprocessableEntity, "Invalid subscription configuration; ask the panel administrator to check the Clash template")
		return
	}

	c.Header("content-disposition", mime.FormatMediaType("attachment", map[string]string{"filename": *account.Username + ".yaml"}))
	c.Header("profile-update-interval", "12")
	c.Header("subscription-userinfo", userInfo)
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", result)
}

// Keep the original token URL valid for existing clients.
func subscriptionPath(password, target string) string {
	path := fmt.Sprintf("/api/auth/subscribe/%s", base64.StdEncoding.EncodeToString([]byte(password)))
	if target == util.ClashVergeTarget {
		path += "?target=" + util.ClashVergeTarget
	}
	return path
}
