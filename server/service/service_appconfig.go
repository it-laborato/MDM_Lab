package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/it-laborato/MDM_Lab/server"
	authz_ctx "github.com/it-laborato/MDM_Lab/server/contexts/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/contexts/license"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mail"
)

// mailError is set when an error performing mail operations
type mailError struct {
	message string
}

func (e mailError) Error() string {
	return fmt.Sprintf("a mail error occurred: %s", e.message)
}

func (e mailError) MailError() []map[string]string {
	return []map[string]string{
		{
			"name":   "base",
			"reason": e.message,
		},
	}
}

func (svc *Service) NewAppConfig(ctx context.Context, p mdmlab.AppConfig) (*mdmlab.AppConfig, error) {
	// skipauth: No user context yet when the app config is first created.
	svc.authz.SkipAuthorization(ctx)

	newConfig, err := svc.ds.NewAppConfig(ctx, &p)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret := svc.config.Packaging.GlobalEnrollSecret
	if secret == "" {
		secret, err = server.GenerateRandomText(mdmlab.EnrollSecretDefaultLength)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
		}
	}
	secrets := []*mdmlab.EnrollSecret{
		{
			Secret: secret,
		},
	}
	err = svc.ds.ApplyEnrollSecrets(ctx, nil, secrets)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "save enroll secret")
	}

	return newConfig, nil
}

func (svc *Service) sendTestEmail(ctx context.Context, config *mdmlab.AppConfig) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return mdmlab.ErrNoContext
	}

	var smtpSettings mdmlab.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}

	testMail := mdmlab.Email{
		Subject: "Hello from MDMlab",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		SMTPSettings: smtpSettings,
		ServerURL:    config.ServerSettings.ServerURL,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return mailError{message: err.Error()}
	}
	return nil
}

func cleanupURL(url string) string {
	return strings.TrimRight(strings.Trim(url, " \t\n"), "/")
}

func (svc *Service) License(ctx context.Context) (*mdmlab.LicenseInfo, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &mdmlab.AppConfig{}, mdmlab.ActionRead); err != nil {
			return nil, err
		}
	}

	lic, _ := license.FromContext(ctx)
	return lic, nil
}

func (svc *Service) SetupRequired(ctx context.Context) (bool, error) {
	users, err := svc.ds.ListUsers(ctx, mdmlab.UserListOptions{ListOptions: mdmlab.ListOptions{Page: 0, PerPage: 1}})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

func (svc *Service) UpdateIntervalConfig(ctx context.Context) (*mdmlab.UpdateIntervalConfig, error) {
	return &mdmlab.UpdateIntervalConfig{
		OSQueryDetail: svc.config.Osquery.DetailUpdateInterval,
		OSQueryPolicy: svc.config.Osquery.PolicyUpdateInterval,
	}, nil
}

func (svc *Service) VulnerabilitiesConfig(ctx context.Context) (*mdmlab.VulnerabilitiesConfig, error) {
	return &mdmlab.VulnerabilitiesConfig{
		DatabasesPath:               svc.config.Vulnerabilities.DatabasesPath,
		Periodicity:                 svc.config.Vulnerabilities.Periodicity,
		CPEDatabaseURL:              svc.config.Vulnerabilities.CPEDatabaseURL,
		CPETranslationsURL:          svc.config.Vulnerabilities.CPETranslationsURL,
		CVEFeedPrefixURL:            svc.config.Vulnerabilities.CVEFeedPrefixURL,
		CurrentInstanceChecks:       svc.config.Vulnerabilities.CurrentInstanceChecks,
		DisableDataSync:             svc.config.Vulnerabilities.DisableDataSync,
		RecentVulnerabilityMaxAge:   svc.config.Vulnerabilities.RecentVulnerabilityMaxAge,
		DisableWinOSVulnerabilities: svc.config.Vulnerabilities.DisableWinOSVulnerabilities,
	}, nil
}

func (svc *Service) LoggingConfig(ctx context.Context) (*mdmlab.Logging, error) {
	conf := svc.config
	logging := &mdmlab.Logging{
		Debug: conf.Logging.Debug,
		Json:  conf.Logging.JSON,
	}

	loggings := []struct {
		plugin string
		target *mdmlab.LoggingPlugin
	}{
		{
			plugin: conf.Osquery.StatusLogPlugin,
			target: &logging.Status,
		},
		{
			plugin: conf.Osquery.ResultLogPlugin,
			target: &logging.Result,
		},
	}

	if conf.Activity.EnableAuditLog {
		loggings = append(loggings, struct {
			plugin string
			target *mdmlab.LoggingPlugin
		}{
			plugin: conf.Activity.AuditLogPlugin,
			target: &logging.Audit,
		})
	}

	for _, lp := range loggings {
		switch lp.plugin {
		case "", "filesystem":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "filesystem",
				Config: mdmlab.FilesystemConfig{
					FilesystemConfig: conf.Filesystem,
				},
			}
		case "kinesis":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "kinesis",
				Config: mdmlab.KinesisConfig{
					Region:       conf.Kinesis.Region,
					StatusStream: conf.Kinesis.StatusStream,
					ResultStream: conf.Kinesis.ResultStream,
					AuditStream:  conf.Kinesis.AuditStream,
				},
			}
		case "firehose":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "firehose",
				Config: mdmlab.FirehoseConfig{
					Region:       conf.Firehose.Region,
					StatusStream: conf.Firehose.StatusStream,
					ResultStream: conf.Firehose.ResultStream,
					AuditStream:  conf.Firehose.AuditStream,
				},
			}
		case "lambda":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "lambda",
				Config: mdmlab.LambdaConfig{
					Region:         conf.Lambda.Region,
					StatusFunction: conf.Lambda.StatusFunction,
					ResultFunction: conf.Lambda.ResultFunction,
					AuditFunction:  conf.Lambda.AuditFunction,
				},
			}
		case "pubsub":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "pubsub",
				Config: mdmlab.PubSubConfig{
					PubSubConfig: conf.PubSub,
				},
			}
		case "stdout":
			*lp.target = mdmlab.LoggingPlugin{Plugin: "stdout"}
		case "kafkarest":
			*lp.target = mdmlab.LoggingPlugin{
				Plugin: "kafkarest",
				Config: mdmlab.KafkaRESTConfig{
					StatusTopic: conf.KafkaREST.StatusTopic,
					ResultTopic: conf.KafkaREST.ResultTopic,
					AuditTopic:  conf.KafkaREST.AuditTopic,
					ProxyHost:   conf.KafkaREST.ProxyHost,
				},
			}
		default:
			return nil, ctxerr.Errorf(ctx, "unrecognized logging plugin: %s", lp.plugin)
		}
	}
	return logging, nil
}

func (svc *Service) EmailConfig(ctx context.Context) (*mdmlab.EmailConfig, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppConfig{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	conf := svc.config
	var email *mdmlab.EmailConfig
	switch conf.Email.EmailBackend {
	case "ses":
		email = &mdmlab.EmailConfig{
			Backend: conf.Email.EmailBackend,
			Config: mdmlab.SESConfig{
				Region:    conf.SES.Region,
				SourceARN: conf.SES.SourceArn,
			},
		}
	default:
		// SES is the only email provider configured as server envs/yaml file, the default implementation, SMTP, is configured via API/UI
		// SMTP config gets its own dedicated section in the AppConfig response
	}

	return email, nil
}
