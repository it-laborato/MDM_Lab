package mail

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
)

func Test_getFromSES(t *testing.T) {
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
			name: "should return properly formatted SMTP from for use in SES",
			args: args{e: mdmlab.Email{
				ServerURL: "https://foobar.mdmlabdm.com",
			}},
			want:    "From: do-not-reply@foobar.mdmlabdm.com\r\n",
			wantErr: assert.NoError,
		},
		{
			name: "should error when we fail to parse mdmlab server url",
			args: args{e: mdmlab.Email{
				ServerURL: "not-a-url",
			}},
			want:    "",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFromSES(tt.args.e)
			if !tt.wantErr(t, err, fmt.Sprintf("getFromSES(%v)", tt.args.e)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getFromSES(%v)", tt.args.e)
		})
	}
}

type mockSESSender struct {
	shouldErr bool
}

func (m mockSESSender) SendRawEmail(input *ses.SendRawEmailInput) (*ses.SendRawEmailOutput, error) {
	if m.shouldErr {
		return nil, errors.New("some error")
	}
	return nil, nil
}

func Test_sesSender_SendEmail(t *testing.T) {
	type fields struct {
		client    mdmlabSESSender
		sourceArn string
	}
	type args struct {
		e mdmlab.Email
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should send email",
			fields: fields{
				client:    mockSESSender{shouldErr: false},
				sourceArn: "foo",
			},
			args: args{e: mdmlab.Email{
				Subject:   "Hello from MDMlab!",
				To:        []string{"foouser@mdmlabdm.com"},
				ServerURL: "https://foobar.mdmlabdm.com",
				Mailer: &SMTPTestMailer{
					BaseURL: "https://localhost:8080",
				},
			}},
			wantErr: assert.NoError,
		},
		{
			name: "should error when email config is nil",
			fields: fields{
				client:    mockSESSender{shouldErr: false},
				sourceArn: "foo",
			},
			args: args{e: mdmlab.Email{
				Subject: "Hello from MDMlab!",
				To:      []string{"foouser@mdmlabdm.com"},
				Mailer: &SMTPTestMailer{
					BaseURL: "https://localhost:8080",
				},
			}},
			wantErr: assert.Error,
		},
		{
			name: "should error when ses client is nil",
			fields: fields{
				client:    nil,
				sourceArn: "foo",
			},
			args: args{e: mdmlab.Email{
				Subject:   "Hello from MDMlab!",
				To:        []string{"foouser@mdmlabdm.com"},
				ServerURL: "https://foobar.mdmlabdm.com",
				Mailer: &SMTPTestMailer{
					BaseURL: "https://localhost:8080",
				},
			}},
			wantErr: assert.Error,
		},
		{
			name: "should error when ses client returns an error",
			fields: fields{
				client:    mockSESSender{shouldErr: true},
				sourceArn: "foo",
			},
			args: args{e: mdmlab.Email{
				Subject:   "Hello from MDMlab!",
				To:        []string{"foouser@mdmlabdm.com"},
				ServerURL: "https://foobar.mdmlabdm.com",
				Mailer: &SMTPTestMailer{
					BaseURL: "https://localhost:8080",
				},
			}},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sesSender{
				client:    tt.fields.client,
				sourceArn: tt.fields.sourceArn,
			}
			tt.wantErr(t, s.SendEmail(tt.args.e), fmt.Sprintf("SendEmail(%v)", tt.args.e))
		})
	}
}
