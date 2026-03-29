package alertengine

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/imysm/db-backup/internal/alertmodel"
	"github.com/imysm/db-backup/internal/alertstorage"
)

// Sender 通知发送者接口
type Sender interface {
	Send(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error
	Type() alertmodel.ChannelType
}

// NotifyService 通知服务
type NotifyService struct {
	senders map[alertmodel.ChannelType]Sender
}

// NewNotifyService 创建通知服务
func NewNotifyService() *NotifyService {
	svc := &NotifyService{
		senders: make(map[alertmodel.ChannelType]Sender),
	}

	// 注册发送者
	svc.senders[alertmodel.ChannelTypeFeishu] = &FeishuSender{}
	svc.senders[alertmodel.ChannelTypeWeCom] = &WeComSender{}
	svc.senders[alertmodel.ChannelTypeDingTalk] = &DingTalkSender{}
	svc.senders[alertmodel.ChannelTypeEmail] = &EmailSender{}

	return svc
}

// SendByChannel 根据渠道类型发送
func (s *NotifyService) SendByChannel(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error {
	sender, ok := s.senders[channel.Type]
	if !ok {
		return fmt.Errorf("未实现的渠道类型: %s", channel.Type)
	}
	return sender.Send(channel, record)
}

// GetSender 获取指定类型的发送者
func (s *NotifyService) GetSender(channelType alertmodel.ChannelType) (Sender, bool) {
	sender, ok := s.senders[channelType]
	return sender, ok
}

// FeishuSender 飞书发送者
type FeishuSender struct{}

func (s *FeishuSender) Type() alertmodel.ChannelType {
	return alertmodel.ChannelTypeFeishu
}

func (s *FeishuSender) Send(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error {
	cfg, err := channel.GetConfig()
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 构建卡片消息
	card := s.buildCard(record)
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 发送请求
	resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("响应状态码异常: %d", resp.StatusCode)
	}

	return nil
}

func (s *FeishuSender) buildCard(record *alertmodel.AlertRecord) map[string]interface{} {
	// 颜色映射
	colors := map[alertmodel.AlertLevel]string{
		alertmodel.AlertLevelP0: "red",
		alertmodel.AlertLevelP1: "orange",
		alertmodel.AlertLevelP2: "yellow",
		alertmodel.AlertLevelP3: "blue",
	}
	color := colors[record.Level]
	if color == "" {
		color = "blue"
	}

	return map[string]interface{}{
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": fmt.Sprintf("🔴 [%s] %s", record.Level, record.Title),
			},
			"template": color,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":  "lark_md",
					"content": fmt.Sprintf("**告警级别**: %s\n**事件类型**: %s\n**触发时间**: %s",
						record.Level, record.EventType, record.TriggeredAt.Format("2006-01-02 15:04:05")),
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":  "lark_md",
					"content": fmt.Sprintf("**详细信息**:\n%s", record.Content),
				},
			},
		},
	}
}

// WeComSender 企业微信发送者
type WeComSender struct{}

func (s *WeComSender) Type() alertmodel.ChannelType {
	return alertmodel.ChannelTypeWeCom
}

func (s *WeComSender) Send(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error {
	cfg, err := channel.GetConfig()
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	content := fmt.Sprintf("### 🔴 [%s] %s\n**事件类型**: %s\n**触发时间**: %s\n**详细信息**: %s",
		record.Level, record.Title, record.EventType,
		record.TriggeredAt.Format("2006-01-02 15:04:05"), record.Content)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("响应状态码异常: %d", resp.StatusCode)
	}

	return nil
}

// DingTalkSender 钉钉发送者
type DingTalkSender struct{}

func (s *DingTalkSender) Type() alertmodel.ChannelType {
	return alertmodel.ChannelTypeDingTalk
}

func (s *DingTalkSender) Send(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error {
	cfg, err := channel.GetConfig()
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	content := fmt.Sprintf("### [%s] %s\n**事件类型**: %s\n**触发时间**: %s\n**详细信息**: %s",
		record.Level, record.Title, record.EventType,
		record.TriggeredAt.Format("2006-01-02 15:04:05"), record.Content)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": fmt.Sprintf("[%s] %s", record.Level, record.Title),
			"text":  content,
		},
	}

	// 如果有签名密钥，添加签名
	if cfg.Secret != "" {
		sign := s.generateSign(cfg.Secret)
		payload["timestamp"] = time.Now().UnixMilli()
		payload["sign"] = sign
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("响应状态码异常: %d", resp.StatusCode)
	}

	return nil
}

func (s *DingTalkSender) generateSign(secret string) string {
	timestamp := time.Now().UnixMilli()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return sign
}

// EmailSender 邮件发送者
type EmailSender struct{}

func (s *EmailSender) Type() alertmodel.ChannelType {
	return alertmodel.ChannelTypeEmail
}

func (s *EmailSender) Send(channel *alertmodel.NotificationChannel, record *alertmodel.AlertRecord) error {
	cfg, err := channel.GetConfig()
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 构建邮件内容
	subject := fmt.Sprintf("[%s] %s", record.Level, record.Title)
	body := s.buildHTML(record)

	// 简单邮件发送
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	to := cfg.To
	if len(to) == 0 {
		return fmt.Errorf("未配置收件人")
	}

	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	if cfg.UseTLS {
		err = s.sendWithTLS(addr, auth, from, to, msg.Bytes())
	} else {
		err = smtp.SendMail(addr, auth, from, to, msg.Bytes())
	}

	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	return nil
}

func (s *EmailSender) buildHTML(record *alertmodel.AlertRecord) string {
	// 颜色映射
	colors := map[alertmodel.AlertLevel]string{
		alertmodel.AlertLevelP0: "#d32f2f",
		alertmodel.AlertLevelP1: "#f57c00",
		alertmodel.AlertLevelP2: "#fbc02d",
		alertmodel.AlertLevelP3: "#1976d2",
	}
	color := colors[record.Level]
	if color == "" {
		color = "#1976d2"
	}

	tmpl := template.Must(template.New("email").Parse(`
<html>
<body style="font-family: Arial, sans-serif;">
<h2 style="color: {{.Color}};">[{{.Level}}] {{.Title}}</h2>
<table style="border-collapse: collapse; width: 100%%;">
  <tr>
    <td style="padding: 8px; border: 1px solid #ddd; width: 120px;"><b>事件类型</b></td>
    <td style="padding: 8px; border: 1px solid #ddd;">{{.EventType}}</td>
  </tr>
  <tr>
    <td style="padding: 8px; border: 1px solid #ddd;"><b>触发时间</b></td>
    <td style="padding: 8px; border: 1px solid #ddd;">{{.TriggeredAt}}</td>
  </tr>
  <tr>
    <td style="padding: 8px; border: 1px solid #ddd;"><b>详细信息</b></td>
    <td style="padding: 8px; border: 1px solid #ddd;">{{.Content}}</td>
  </tr>
</table>
<p style="color: #666; font-size: 12px;">此邮件由 db-backup 系统自动发送</p>
</body>
</html>
`))

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Level":       record.Level,
		"Title":       record.Title,
		"EventType":   record.EventType,
		"TriggeredAt": record.TriggeredAt.Format("2006-01-02 15:04:05"),
		"Content":     record.Content,
		"Color":       color,
	}
	tmpl.Execute(&buf, data)
	return buf.String()
}

func (s *EmailSender) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// 简化实现，实际应该使用 crypto/tls
	return smtp.SendMail(addr, auth, from, to, msg)
}

// AlertNotifier 告警通知器
type AlertNotifier struct {
	notifyService  *NotifyService
	recordStorage *alertstorage.RecordStorage
	channelStorage *alertstorage.ChannelStorage
}

// NewAlertNotifier 创建告警通知器
func NewAlertNotifier(recordStorage *alertstorage.RecordStorage, channelStorage *alertstorage.ChannelStorage) *AlertNotifier {
	return &AlertNotifier{
		notifyService:  NewNotifyService(),
		recordStorage:  recordStorage,
		channelStorage: channelStorage,
	}
}

// NotifyAlert 发送告警通知
func (n *AlertNotifier) NotifyAlert(record *alertmodel.AlertRecord, channels []*alertmodel.NotificationChannel) error {
	var lastErr error

	for _, channel := range channels {
		if !channel.IsEnabled() {
			continue
		}

		err := n.notifyService.SendByChannel(channel, record)
		now := time.Now()

		// 更新发送记录
		notifRecords, _ := n.recordStorage.GetNotificationRecords(record.ID)
		for i := range notifRecords {
			nr := &notifRecords[i]
			if nr.ChannelID == channel.ID && nr.Status == "pending" {
				nr.Status = "sent"
				nr.SentAt = &now
				if err != nil {
					nr.Status = "failed"
					nr.ErrorMsg = err.Error()
				}
				n.recordStorage.UpdateNotificationRecord(nr)
				break
			}
		}

		if err != nil {
			lastErr = err
			// 更新渠道失败计数
			n.channelStorage.IncrementFailedCount(channel.ID, err.Error())
			continue
		}

		// 更新渠道成功计数
		n.channelStorage.UpdateLastSent(channel.ID)
	}

	return lastErr
}
