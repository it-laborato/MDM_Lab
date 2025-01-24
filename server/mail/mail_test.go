package mail

import (
	"fmt"
	"os"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFunctions = [...]func(*testing.T, mdmlab.MailService){
	testSMTPPlainAuth,
	testSMTPPlainAuthInvalidCreds,
	testSMTPSkipVerify,
	testSMTPNoAuthWithTLS,
	testMailTest,
}

func TestCanSendMail(t *testing.T) {
	settings := mdmlab.SMTPSettings{
		SMTPConfigured:           true,
		SMTPAuthenticationType:   mdmlab.AuthTypeNameUserNamePassword,
		SMTPAuthenticationMethod: mdmlab.AuthMethodNamePlain,
		SMTPUserName:             "mailpit-username",
		SMTPPassword:             "mailpit-password",
		SMTPEnableTLS:            false,
		SMTPVerifySSLCerts:       false,
		SMTPEnableStartTLS:       false,
		SMTPPort:                 1026,
		SMTPServer:               "localhost",
		SMTPSenderAddress:        "test@example.com",
	}

	r, err := NewService(config.TestConfig())
	require.NoError(t, err)
	require.True(t, r.CanSendEmail(settings))
	require.False(t, r.CanSendEmail(mdmlab.SMTPSettings{}))
}

func TestMail(t *testing.T) {
	// This mail test requires mailhog unauthenticated running on localhost:1025
	// and mailpit running on localhost:1026.
	if _, ok := os.LookupEnv("MAIL_TEST"); !ok {
		t.Skip("Mail tests are disabled")
	}

	for _, f := range testFunctions {
		r, err := NewService(config.TestConfig())
		require.NoError(t, err)

		t.Run(test.FunctionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}

func testSMTPPlainAuth(t *testing.T, mailer mdmlab.MailService) {
	mail := mdmlab.Email{
		Subject: "smtp plain auth",
		To:      []string{"john@mdmlab.co"},
		SMTPSettings: mdmlab.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   mdmlab.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: mdmlab.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "mailpit-password",
			SMTPEnableTLS:            false,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       false,
			SMTPPort:                 1026,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPPlainAuthInvalidCreds(t *testing.T, mailer mdmlab.MailService) {
	mail := mdmlab.Email{
		Subject: "smtp plain auth with invalid credentials",
		To:      []string{"john@mdmlab.co"},
		SMTPSettings: mdmlab.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   mdmlab.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: mdmlab.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "wrong",
			SMTPEnableTLS:            false,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       false,
			SMTPPort:                 1026,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Error(t, err)
}

func testSMTPSkipVerify(t *testing.T, mailer mdmlab.MailService) {
	mail := mdmlab.Email{
		Subject: "skip verify",
		To:      []string{"john@mdmlab.co"},
		SMTPSettings: mdmlab.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   mdmlab.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: mdmlab.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "mailpit-password",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 1027,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPNoAuthWithTLS(t *testing.T, mailer mdmlab.MailService) {
	mail := mdmlab.Email{
		Subject: "no auth",
		To:      []string{"bob@foo.com"},
		SMTPSettings: mdmlab.SMTPSettings{
			SMTPConfigured:         true,
			SMTPAuthenticationType: mdmlab.AuthTypeNameNone,
			SMTPEnableTLS:          true,
			SMTPVerifySSLCerts:     true,
			SMTPEnableStartTLS:     true,
			SMTPPort:               1027,
			SMTPServer:             "localhost",
			SMTPSenderAddress:      "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testMailTest(t *testing.T, mailer mdmlab.MailService) {
	mail := mdmlab.Email{
		Subject: "test tester",
		To:      []string{"bob@foo.com"},
		SMTPSettings: mdmlab.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   mdmlab.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: mdmlab.AuthMethodNamePlain,
			SMTPUserName:             "foo",
			SMTPPassword:             "bar",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       true,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 1027,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}
	err := Test(mailer, mail)
	assert.Nil(t, err)
}

func TestTemplateProcessor(t *testing.T) {
	mailer := PasswordResetMailer{
		BaseURL: "https://localhost.com:8080",
		Token:   "12345",
	}

	out, err := mailer.Message()
	require.Nil(t, err)
	assert.NotNil(t, out)
}

func Test_getFrom(t *testing.T) {
	type args struct {
		e mdmlab.Email
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return SMTP formatted From string",
			args: args{
				e: mdmlab.Email{
					SMTPSettings: mdmlab.SMTPSettings{
						SMTPSenderAddress: "foo@bar.com",
					},
				},
			},
			want:    "From: foo@bar.com\r\n",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFrom(tt.args.e)
			if !tt.wantErr(t, err, fmt.Sprintf("getFrom(%v)", tt.args.e)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getFrom(%v)", tt.args.e)
		})
	}
}
