package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

type aliyunSender struct {
	client *dysmsapi.Client
	conf   *conf.Data_Sms
	log    *log.Helper
}

func NewAliyunSender(c *conf.Data, logger log.Logger) Sender {
	// 1. 使用官方推荐的凭据初始化方式
	cred, err := credentials.NewCredential(&credentials.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(c.Sms.AccessKey),
		AccessKeySecret: tea.String(c.Sms.AccessSecret),
	})
	if err != nil {
		panic(fmt.Sprintf("初始化阿里云凭据失败: %v", err))
	}

	config := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String("dysmsapi.aliyuncs.com"),
	}

	client, err := dysmsapi.NewClient(config)
	if err != nil {
		panic(fmt.Sprintf("创建阿里云短信客户端失败: %v", err))
	}

	return &aliyunSender{
		client: client,
		conf:   c.Sms,
		log:    log.NewHelper(logger),
	}
}

func (s *aliyunSender) Send(ctx context.Context, phone string, template string, params map[string]string) error {
	templateCode, ok := s.conf.TemplateMapping[template]
	if !ok || templateCode == "" {
		return ErrorTemplateNotConfigured
	}

	jsonParams, _ := json.Marshal(params)

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(s.conf.SignName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(jsonParams)),
	}

	// 使用 RuntimeOptions 配置超时
	runtime := &util.RuntimeOptions{
		ConnectTimeout: tea.Int(5000), // 5秒连接超时
		ReadTimeout:    tea.Int(5000), // 5秒读取超时
	}

	// 按照官方示例处理 Tea 框架的 Panic 和 Error
	var sendErr error
	tryErr := func() (e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				e = r
			}
		}()

		resp, err := s.client.SendSmsWithOptions(request, runtime)
		if err != nil {
			return err
		}

		// 注意：阿里云返回 200 不代表短信发送成功（例如欠费、频率限制）
		// 必须判断 Body 中的 Code
		if tea.StringValue(resp.Body.Code) != "OK" {
			return fmt.Errorf("阿里云业务报错: %s - %s",
				tea.StringValue(resp.Body.Code),
				tea.StringValue(resp.Body.Message))
		}

		s.log.Infof("短信发送成功: RequestId=%s", tea.StringValue(resp.Body.RequestId))
		return nil
	}()

	if tryErr != nil {
		sendErr = tryErr
		// 重点优化：解析阿里云特有的 SDKError 以获取诊断建议
		var sdkErr *tea.SDKError
		if errors.As(tryErr, &sdkErr) {
			s.log.Errorf("阿里云SDK错误: %s", tea.StringValue(sdkErr.Message))

			// 解析诊断数据中的 Recommend
			var diagData interface{}
			if err := json.Unmarshal([]byte(tea.StringValue(sdkErr.Data)), &diagData); err == nil {
				if m, ok := diagData.(map[string]interface{}); ok {
					if recommend, ok := m["Recommend"]; ok {
						s.log.Warnf("阿里云诊断建议: %v", recommend)
					}
				}
			}
		}
	}

	return sendErr
}
