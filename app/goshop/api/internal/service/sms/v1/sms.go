package v1

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"goshop/app/pkg/options"
	"math/big"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

type SmsSrv interface {
	//
	// SendSms
	//  @Description: 发送短信验证码
	//  @param ctx
	//  @param mobile: 手机号码
	//  @param tpc: template code 消息模板编号
	//  @param tp: template param消息参数
	//  @return error
	//
	SendSms(ctx context.Context, mobile string, tpc, tp string) error
}

func GenerateSmsCode(width int) (string, error) {
	//生成width长度的短信验证码
	if width <= 0 {
		return "", nil
	}

	maxLen := big.NewInt(10)
	var sb strings.Builder
	sb.Grow(width)
	for i := 0; i < width; i++ {
		n, err := crand.Int(crand.Reader, maxLen)
		if err != nil {
			return "", fmt.Errorf("generate sms code: %w", err)
		}
		sb.WriteByte(byte('0' + n.Int64()))
	}
	return sb.String(), nil
}

func (s *smsService) SendSms(ctx context.Context, mobile string, tpc, tp string) error {
	if s == nil || s.smsOpts == nil {
		return errors.New("sms options missing")
	}
	if s.smsOpts.APIKey == "" || s.smsOpts.APISecret == "" {
		return errors.New("sms credentials missing")
	}

	client, err := dysmsapi.NewClientWithAccessKey("cn-beijing", s.smsOpts.APIKey, s.smsOpts.APISecret)
	if err != nil {
		return fmt.Errorf("create sms client: %w", err)
	}
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "cn-beijing"
	request.QueryParams["PhoneNumbers"] = mobile //手机号
	request.QueryParams["SignName"] = "慕学在线"     //阿里云验证过的项目名 自己设置
	request.QueryParams["TemplateCode"] = tpc    //阿里云的短信模板号 自己设置
	request.QueryParams["TemplateParam"] = tp    //短信模板中的验证码内容 自己生成   之前试过直接返回，但是失败，加上code成功。
	if _, err := client.ProcessCommonRequest(request); err != nil {
		return fmt.Errorf("send sms: %w", err)
	}
	return nil
}

type smsService struct {
	smsOpts *options.SmsOptions
}

func NewSmsService(smsOpts *options.SmsOptions) SmsSrv {
	return &smsService{smsOpts: smsOpts}
}
