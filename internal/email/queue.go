package email

import (
	"github.com/ArtalkJS/ArtalkGo/internal/config"
	"github.com/ArtalkJS/ArtalkGo/internal/query"
	"github.com/sirupsen/logrus"
)

// Email Queue
var emailCh chan Email

func InitQueue() {
	if emailCh != nil {
		emailCh = make(chan Email) // TODO: add size limit
	}

	go func() {
		for {
			select {
			case email := <-emailCh:
				sender := NewSender(config.Instance.Email.SendType)

				if sender.Send(email) { // 发送成功
					if email.LinkedNotify != nil {
						// 标记关联评论邮件发送状态
						if err := query.NotifySetEmailed(email.LinkedNotify); err != nil {
							logrus.Errorf("[EMAIL] 标记关联评论邮件发送状态失败: %s", err)
							continue
						}
					}
				}
			}
		}
	}()
}

func AddToQueue(email Email) {
	emailCh <- email
}
