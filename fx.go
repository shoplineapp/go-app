package app

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx/fxevent"
)

var fxLogger = AppLogger{}

type AppLogger struct {
	fxevent.Logger
}

func (AppLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		logrus.Debug(fmt.Sprintf("HOOK OnStart\t\t%s executing (caller: %s)", e.FunctionName, e.CallerName))
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("HOOK OnStart\t\t%s called by %s failed in %s: %v", e.FunctionName, e.CallerName, e.Runtime, e.Err))
		} else {
			logrus.Debug(fmt.Sprintf("HOOK OnStart\t\t%s called by %s ran successfully in %s", e.FunctionName, e.CallerName, e.Runtime))
		}
	case *fxevent.OnStopExecuting:
		logrus.Debug(fmt.Sprintf("HOOK OnStop\t\t%s executing (caller: %s)", e.FunctionName, e.CallerName))
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("HOOK OnStop\t\t%s called by %s failed in %s: %v", e.FunctionName, e.CallerName, e.Runtime, e.Err))
		} else {
			logrus.Debug(fmt.Sprintf("HOOK OnStop\t\t%s called by %s ran successfully in %s", e.FunctionName, e.CallerName, e.Runtime))
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Failed to supply %v: %v", e.TypeName, e.Err))
		} else if e.ModuleName != "" {
			logrus.Info(fmt.Sprintf("SUPPLY %v from module %q", e.TypeName, e.ModuleName))
		} else {
			logrus.Info(fmt.Sprintf("SUPPLY %v", e.TypeName))
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			if e.ModuleName != "" {
				logrus.Info(fmt.Sprintf("PROVIDE plugin %v <= from module %q", rtype, e.ModuleName))
			} else {
				logrus.Info(fmt.Sprintf("PROVIDE plugin %v", rtype))
			}
		}
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Error after options were applied: %v", e.Err))
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			if e.ModuleName != "" {
				logrus.Debug(fmt.Sprintf("DECORATE %v <= %v from module %q", rtype, e.DecoratorName, e.ModuleName))
			} else {
				logrus.Debug(fmt.Sprintf("DECORATE %v <= %v", rtype, e.DecoratorName))
			}
		}
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Error after options were applied: %v", e.Err))
		}
	case *fxevent.Invoking:
		if e.ModuleName != "" {
			logrus.Debug(fmt.Sprintf("INVOKE %s from module %q", e.FunctionName, e.ModuleName))
		} else {
			logrus.Debug(fmt.Sprintf("INVOKE %s", e.FunctionName))
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Failed to invoke %v called from:\n%+vFailed: %v", e.FunctionName, e.Trace, e.Err))
		}
	case *fxevent.Stopping:
		logrus.Warn(fmt.Sprintf("Received %s", strings.ToUpper(e.Signal.String())))
	case *fxevent.Stopped:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Failed to stop cleanly: %v", e.Err))
		}
	case *fxevent.RollingBack:
		logrus.Error(fmt.Sprintf("Start failed, rolling back: %v", e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Couldn't roll back cleanly: %v", e.Err))
		}
	case *fxevent.Started:
		if e.Err != nil {
			logrus.Error(fmt.Sprintf("Failed to start: %v", e.Err))
		} else {
			logrus.Info("Application RUNNING")
		}
	}
}

func (AppLogger) String() string {
	return "InternalLogger"
}
