package util

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/smtp"
	"strings"
	"time"

	"forxi.cn/forxi-go/app/config"
)

// GenerateVerificationCode 生成6位随机数字验证码
func GenerateVerificationCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += fmt.Sprintf("%d", n)
	}
	return code
}

// 邮件模板
const (
	registerEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>邮箱验证码</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
        <div style="background-color: #ffffff; padding: 30px; border-radius: 10px;">
            <h2 style="color: #333333; margin-top: 0;">欢迎注册 Forxi</h2>
            <p style="color: #666666; font-size: 16px; line-height: 1.6;">
                您正在进行邮箱验证，您的验证码为：
            </p>
            <div style="background-color: #f8f8f8; padding: 20px; border-radius: 5px; text-align: center; margin: 20px 0;">
                <span style="font-size: 32px; font-weight: bold; color: #4CAF50; letter-spacing: 5px;">{{CODE}}</span>
            </div>
            <p style="color: #999999; font-size: 14px;">
                验证码有效期为10分钟，请尽快完成验证。
            </p>
            <p style="color: #999999; font-size: 14px;">
                如果这不是您的操作，请忽略此邮件。
            </p>
        </div>
        <div style="text-align: center; margin-top: 20px; color: #999999; font-size: 12px;">
            <p>此邮件由系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`

	resetPasswordEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>密码重置</title>
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
        <div style="background-color: #ffffff; padding: 30px; border-radius: 10px;">
            <h2 style="color: #333333; margin-top: 0;">密码重置</h2>
            <p style="color: #666666; font-size: 16px; line-height: 1.6;">
                您正在进行密码重置，请点击以下链接完成密码重置：
            </p>
            <div style="margin: 20px 0;">
                <a href="{{RESET_LINK}}" style="display: inline-block; background-color: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-size: 16px;">重置密码</a>
            </div>
            <p style="color: #999999; font-size: 14px;">
                链接有效期为15分钟，请尽快完成密码重置。
            </p>
            <p style="color: #999999; font-size: 14px;">
                如果这不是您的操作，请立即修改您的账户密码以确保安全。
            </p>
        </div>
        <div style="text-align: center; margin-top: 20px; color: #999999; font-size: 12px;">
            <p>此邮件由系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`
)

// encodeSubject 编码邮件主题（支持中文）
func encodeSubject(subject string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
}

// SendVerificationCode 发送验证码邮件
// purpose: register - 注册, reset_password - 密码重置
// resetLink: 可选，用于密码重置邮件中的链接
func SendVerificationCode(cfg *config.EmailConfig, to, code, purpose string, resetLink ...string) error {
	// 选择邮件模板
	var template string
	var subject string

	switch purpose {
	case "register":
		template = registerEmailTemplate
		subject = "【Forxi Auth】邮箱验证码"
	case "reset_password":
		template = resetPasswordEmailTemplate
		subject = "【Forxi Auth】密码重置"
	default:
		return fmt.Errorf("unknown email purpose: %s", purpose)
	}

	// 替换验证码
	body := strings.ReplaceAll(template, "{{CODE}}", code)

	// 如果有重置链接，替换链接
	if len(resetLink) > 0 && resetLink[0] != "" {
		body = strings.ReplaceAll(body, "{{RESET_LINK}}", resetLink[0])
	}

	// 生成Message-ID
	timestamp := time.Now().Unix()
	messageID := fmt.Sprintf("<%d.%s@%s>", timestamp, generateRandomString(16), cfg.SMTPHost)

	// 构造完整的邮件头（按RFC 5322标准）
	var message strings.Builder
	message.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", cfg.FromName, cfg.FromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(subject)))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("\r\n")

	// 使用Base64编码邮件正文
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	// 按照RFC 2045规范，每76个字符换行
	for i := 0; i < len(encodedBody); i += 76 {
		end := i + 76
		if end > len(encodedBody) {
			end = len(encodedBody)
		}
		message.WriteString(encodedBody[i:end])
		message.WriteString("\r\n")
	}

	// 服务器地址
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// 对于465端口，使用SSL直接连接
	if cfg.SMTPPort == 465 {
		return sendMailWithSSL(addr, cfg.Username, cfg.Password, cfg.SMTPHost, cfg.FromEmail, []string{to}, []byte(message.String()))
	}

	// 对于其他端口（如587），使用标准SMTP
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	err := smtp.SendMail(addr, auth, cfg.FromEmail, []string{to}, []byte(message.String()))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// sendMailWithSSL 使用SSL发送邮件（用于465端口）
func sendMailWithSSL(addr, username, password, host, from string, to []string, msg []byte) error {
	// 创建TLS配置
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	// 建立TLS连接
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// 发送EHLO命令，使用正确的主机名
	if err = client.Hello(host); err != nil {
		return fmt.Errorf("failed to send EHLO: %w", err)
	}

	// 认证
	auth := smtp.PlainAuth("", username, password, host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// 设置发件人
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// 设置收件人
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	// 退出
	return client.Quit()
}
