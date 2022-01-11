package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/v14/internal/bootstrap"
	"gitlab.com/gitlab-org/gitaly/v14/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/v14/internal/testhelper/testcfg"
)

// TestStolenPid tests for regressions in https://gitlab.com/gitlab-org/gitaly/issues/1661
func TestStolenPid(t *testing.T) {
	tempDir := testhelper.TempDir(t)

	pidFile, err := os.Create(filepath.Join(tempDir, "pidfile"))
	require.NoError(t, err)

	ctx, cancel := testhelper.Context()
	defer cancel()

	cmd := exec.CommandContext(ctx, "tail", "-f")
	require.NoError(t, cmd.Start())
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	_, err = pidFile.WriteString(strconv.Itoa(cmd.Process.Pid))
	require.NoError(t, err)
	require.NoError(t, pidFile.Close())

	tail, err := findProcess(pidFile.Name())
	require.NoError(t, err)
	require.NotNil(t, tail)
	require.Equal(t, cmd.Process.Pid, tail.Pid)

	t.Run("stolen", func(t *testing.T) {
		require.False(t, isExpectedProcess(tail, "/path/to/gitaly"))
	})

	t.Run("not stolen", func(t *testing.T) {
		require.True(t, isExpectedProcess(tail, "/path/to/tail"))
	})
}

func TestFindProcess(t *testing.T) {
	t.Run("nonexistent PID file", func(t *testing.T) {
		_, err := findProcess("does-not-exist")
		require.True(t, os.IsNotExist(err))
	})

	t.Run("PID file with garbage", func(t *testing.T) {
		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, []byte("garbage"), 0o644))

		_, err := findProcess(path)
		_, expectedErr := strconv.Atoi("garbage")
		require.Equal(t, expectedErr, err)
	})

	t.Run("nonexistent process", func(t *testing.T) {
		// The below PID can exist, but chances are sufficiently low to hopefully not matter
		// in practice.
		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, []byte("7777777"), 0o644))

		// The process isn't alive, so we expect neither an error nor a process to be
		// returned.
		process, err := findProcess(path)
		require.Nil(t, process)
		require.Nil(t, err)
	})

	t.Run("running process", func(t *testing.T) {
		ctx, cancel := testhelper.Context()
		defer cancel()

		executable := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "noop"), []byte(
			`#!/usr/bin/env bash
			while true
			do
				:
			done
		`))

		cmd := exec.CommandContext(ctx, executable)
		require.NoError(t, cmd.Start())
		defer func() {
			cancel()
			require.Error(t, cmd.Wait())
		}()

		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, []byte(strconv.FormatInt(int64(cmd.Process.Pid), 10)), 0o644))

		process, err := findProcess(path)
		require.NotNil(t, process)
		require.Equal(t, cmd.Process.Pid, process.Pid)
		require.Equal(t, nil, err)
	})
}

func TestIsRecoverable(t *testing.T) {
	_, numericError := strconv.Atoi("")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "file doesn't exist",
			err:  os.ErrNotExist,
			want: true,
		},
		{
			name: "numeric error",
			err:  numericError,
			want: true,
		},
		{
			name: "generic error",
			err:  errors.New("generic error"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecoverable(tt.err); got != tt.want {
				t.Errorf("isRecoverable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadPIDFile(t *testing.T) {
	t.Run("nonexistent", func(t *testing.T) {
		_, err := readPIDFile("does-not-exist")
		require.True(t, os.IsNotExist(err))
	})

	t.Run("empty", func(t *testing.T) {
		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, nil, 0o644))
		_, err := readPIDFile(path)
		_, expectedErr := strconv.Atoi("")
		require.Equal(t, expectedErr, err)
	})

	t.Run("invalid contents", func(t *testing.T) {
		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, []byte("invalid"), 0o644))
		_, err := readPIDFile(path)
		_, expectedErr := strconv.Atoi("invalid")
		require.Equal(t, expectedErr, err)
	})

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(path, []byte("12345"), 0o644))
		pid, err := readPIDFile(path)
		require.NoError(t, err)
		require.Equal(t, 12345, pid)
	})
}

func TestIsExpectedProcess(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	executable := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "noop"), []byte(
		`#!/usr/bin/env bash
		while true
		do
			:
		done
	`))

	cmd := exec.CommandContext(ctx, executable)
	require.NoError(t, cmd.Start())

	require.False(t, isExpectedProcess(cmd.Process, "does not match"))
	require.True(t, isExpectedProcess(cmd.Process, "bash"))

	cancel()
	require.Error(t, cmd.Wait())
}

func TestIsProcessAlive(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	executable := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "noop"), []byte(
		`#!/usr/bin/env bash
		while true
		do
			:
		done
	`))

	cmd := exec.CommandContext(ctx, executable)
	require.NoError(t, cmd.Start())
	require.True(t, isProcessAlive(cmd.Process))

	// The process will be considered alive as long as it hasn't been reaped yet.
	require.NoError(t, cmd.Process.Kill())
	require.True(t, isProcessAlive(cmd.Process))

	require.Error(t, cmd.Wait())
	require.False(t, isProcessAlive(cmd.Process))

	// And now let's check with a nonexistent process. FindProcess never returns an error on
	// Unix systems even if the process doesn't exist, so this is fine.
	process, err := os.FindProcess(77777777)
	require.NoError(t, err)
	require.False(t, isProcessAlive(process))
}

func TestRun(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	binary := testcfg.BuildGitalyWrapper(t, testcfg.Build(t))

	t.Run("missing arguments", func(t *testing.T) {
		output, err := exec.CommandContext(ctx, binary).CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "usage: ")
	})

	t.Run("missing PID file envvar", func(t *testing.T) {
		output, err := exec.CommandContext(ctx, binary, "binary").CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "missing pid file ENV variable")
	})

	t.Run("invalid executable", func(t *testing.T) {
		pidPath := filepath.Join(testhelper.TempDir(t), "pid")

		cmd := exec.CommandContext(ctx, binary, "does-not-exist")
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", bootstrap.EnvPidFile, pidPath))

		output, err := cmd.CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "executable file not found in $PATH")
	})

	t.Run("adopting executable", func(t *testing.T) {
		script := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "script"), []byte(
			`#!/usr/bin/env bash
			sleep 1
		`))

		scriptCmd := exec.CommandContext(ctx, script)
		require.NoError(t, scriptCmd.Start())
		go func() {
			require.NoError(t, scriptCmd.Wait())
		}()

		// Write the PID of the running process into the PID file. As a result, it should
		// get adopted by gitaly-wrapper, which means it wouldn't try to execute it anew.
		pidPath := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(pidPath, []byte(strconv.FormatInt(int64(scriptCmd.Process.Pid), 10)), 0o644))

		var output bytes.Buffer
		// Run gitaly-script with a binary path whose basename matches, but which ultimately
		// doesn't exist. This proves that it doesn't try to execute the script again.
		cmd := exec.CommandContext(ctx, binary, "/does/not/exist/bash")
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", bootstrap.EnvPidFile, pidPath))
		cmd.Stdout = &output
		cmd.Stderr = &output
		require.NoError(t, cmd.Start())

		// The wrapper shall exit when the wrapped script does.
		require.NoError(t, cmd.Wait(), output.String())
		require.Contains(t, output.String(), "adopting a process")
		require.Contains(t, output.String(), "wrapper for process shutting down")
	})

	t.Run("spawning executable", func(t *testing.T) {
		script := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "script"), []byte(
			`#!/usr/bin/env bash
			echo "I have been executed"
		`))

		pidPath := filepath.Join(testhelper.TempDir(t), "pid")

		cmd := exec.CommandContext(ctx, binary, script)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", bootstrap.EnvPidFile, pidPath))
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		require.Contains(t, string(output), "spawning a process")
		require.Contains(t, string(output), "I have been executed")

		require.NoFileExists(t, pidPath)
	})

	t.Run("spawning executable with dead process", func(t *testing.T) {
		script := testhelper.WriteExecutable(t, filepath.Join(testhelper.TempDir(t), "script"), []byte(
			`#!/usr/bin/env bash
			echo "I have been executed"
		`))

		pidPath := filepath.Join(testhelper.TempDir(t), "pid")
		require.NoError(t, os.WriteFile(pidPath, []byte("12345"), 0o644))

		cmd := exec.CommandContext(ctx, binary, script)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", bootstrap.EnvPidFile, pidPath))
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		require.Contains(t, string(output), "spawning a process")
		require.Contains(t, string(output), "I have been executed")
	})
}
