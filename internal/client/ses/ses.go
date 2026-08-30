// Package ses envuelve el cliente de Amazon SES v2 para envio de emails
// transaccionales (password recovery, welcome, etc). Usa el IAM role del EC2
// (sin API keys). Region us-east-1.
package ses

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// Client wrappea sesv2.Client con defaults y helpers.
type Client struct {
	sdk         *sesv2.Client
	fromAddress string // Ej. "Exwood <noreply@exwood.dev>"
}

// New construye un cliente listo para usar.
// Envs esperadas:
//   - AWS_REGION (default us-east-1)
//   - SES_FROM_ADDRESS (ej. "Exwood <noreply@exwood.dev>")
//     Si no esta setada, retorna error — no queremos mandar mails sin remitente identificable.
func New(ctx context.Context) (*Client, error) {
	from := os.Getenv("SES_FROM_ADDRESS")
	if from == "" {
		return nil, fmt.Errorf("SES_FROM_ADDRESS no configurado")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("cargando config aws: %w", err)
	}

	return &Client{
		sdk:         sesv2.NewFromConfig(cfg),
		fromAddress: from,
	}, nil
}

// SendEmail envia un email transaccional. HTML + fallback text.
// Retorna el MessageId de SES si tuvo exito, o error.
func (c *Client) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) (string, error) {
	out, err := c.sdk.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(c.fromAddress),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(htmlBody),
						Charset: aws.String("UTF-8"),
					},
					Text: &types.Content{
						Data:    aws.String(textBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("ses send: %w", err)
	}
	return aws.ToString(out.MessageId), nil
}
