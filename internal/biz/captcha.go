package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/mojocn/base64Captcha"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
)

var (
	ErrorImageCaptchaEmpty        = errors.BadRequest("IMAGE_CAPTCHA_EMPTY", "验证码不能为空")
	ErrorImageCaptchaVerifyFailed = errors.BadRequest("IMAGE_CAPTCHA_VERIFY_FAILED", "图片验证码错误")
)

type CaptchaUseCase struct {
	store base64Captcha.Store
	log   *log.Helper
}

func NewCaptchaUseCase(store base64Captcha.Store, logger log.Logger) *CaptchaUseCase {
	return &CaptchaUseCase{
		store: store,
		log:   log.NewHelper(logger),
	}
}

// Generate 生成验证码
func (uc *CaptchaUseCase) Generate(ctx context.Context) (id, b64 string, err error) {

	// 验证码配置

	// 1. 强干扰数字版
	//driver := base64Captcha.NewDriverDigit(80, 240, 6, 0.8, 100)

	// 2. 清爽数字版
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.3, 10)

	// 3. 字母数字混合版
	//driver := base64Captcha.NewDriverString(
	//	80,  // 高度
	//	240, // 宽度
	//	2,   // 干扰线数量
	//	base64Captcha.OptionShowSlimeLine|base64Captcha.OptionShowHollowLine, // 干扰线类型：粘液线+空心线
	//	6, // 长度
	//	"23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ", // 去除了易混淆字符的库
	//	nil, // 背景颜色，nil 为随机
	//	nil, // 字体，nil 为默认
	//	nil, // 自定义字体路径
	//)

	// 4. 算数版
	//driver := base64Captcha.NewDriverMath(
	//	80,                                // 高度
	//	240,                               // 宽度
	//	0,                                 // 噪点数量（算术题通常靠干扰线和形状，不需要太多噪点）
	//	base64Captcha.OptionShowSlimeLine, // 干扰线类型
	//	nil,                               // 背景颜色
	//	nil,                               // 字体
	//	nil,                               // 自定义字体路径
	//)

	cp := base64Captcha.NewCaptcha(driver, uc.store)

	id, b64, answer, err := cp.Generate()
	if err != nil {
		return "", "", err
	}

	// 如果是调试模式，将答案注入 Context
	if debug.IsDebug() {
		if info, ok := debug.FromContext(ctx); ok {
			info["captcha_answer"] = answer
		}
	}

	return id, b64, nil
}

// Verify 验证验证码
func (uc *CaptchaUseCase) Verify(ctx context.Context, id, answer string) error {
	if id == "" || answer == "" {
		return ErrorImageCaptchaEmpty
	}

	// 校验并自动删除（防止重放攻击）
	if !uc.store.Verify(id, answer, true) {
		return ErrorImageCaptchaVerifyFailed
	}
	return nil
}
