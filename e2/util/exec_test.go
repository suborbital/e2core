package util

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		runner  func() (CommandRunner, *bytes.Buffer)
		want    string
		wantErr assert.ErrorAssertionFunc
		wantBuf []byte
	}{

		{
			name: "writes to a supplied io.Writer",
			cmd:  "echo 'the mitochondria is the powerhouse of the cell'",
			runner: func() (CommandRunner, *bytes.Buffer) {
				buf := new(bytes.Buffer)
				return NewCommandLineExecutor(NormalOutput, buf), buf
			},
			want:    "the mitochondria is the powerhouse of the cell\n",
			wantErr: assert.NoError,
			wantBuf: []byte("the mitochondria is the powerhouse of the cell\n"),
		},
		{
			name: "writes nothing to a supplied io.Writer when silent",
			cmd:  "echo 'the mitochondria is the powerhouse of the cell'",
			runner: func() (CommandRunner, *bytes.Buffer) {
				buf := new(bytes.Buffer)
				return NewCommandLineExecutor(SilentOutput, buf), buf
			},
			want:    "the mitochondria is the powerhouse of the cell\n",
			wantErr: assert.NoError,
			wantBuf: nil,
		},
		{
			name: "accepts a nil io.Writer",
			cmd:  "echo 'the mitochondria is the powerhouse of the cell'",
			runner: func() (CommandRunner, *bytes.Buffer) {
				return NewCommandLineExecutor(SilentOutput, nil), new(bytes.Buffer)
			},
			want:    "the mitochondria is the powerhouse of the cell\n",
			wantErr: assert.NoError,
			wantBuf: nil,
		},
		{
			name: "performs with the default executor",
			cmd:  "echo 'the mitochondria is the powerhouse of the cell'",
			runner: func() (CommandRunner, *bytes.Buffer) {
				return Command, new(bytes.Buffer)
			},
			want:    "the mitochondria is the powerhouse of the cell\n",
			wantErr: assert.NoError,
			wantBuf: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, buf := tt.runner()
			got, err := runner.Run(tt.cmd)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantBuf, buf.Bytes())
		})
	}
}
