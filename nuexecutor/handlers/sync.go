package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/suborbital/e2core/nuexecutor/worker"
)

func Sync(_ chan<- worker.Job) echo.HandlerFunc {

	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, "woo")
		// if err := w.pool.UseInstance(ctx, spanCtx, func(ctx context.Context, instance *instance.Instance, ident int32) {
		// 	_, span := tracing.Tracer.Start(ctx, "instance function")
		// 	defer span.End()
		//
		// 	inPointer, writeErr := instance.WriteMemory(jobBytes)
		// 	if writeErr != nil {
		// 		runErr = errors.Wrap(writeErr, "failed to instance.writeMemory")
		// 		return
		// 	}
		//
		// 	span.AddEvent("instance.Call run_e")
		// 	// execute the module's Run function, passing the input data and ident
		// 	// set runErr but don't return because the ExecutionResult error should also be grabbed
		// 	_, callErr = instance.Call("run_e", inPointer, int32(len(jobBytes)), ident)
		//
		// 	// get the results from the instance
		// 	output, runErr = instance.ExecutionResult()
		//
		// 	// deallocate the memory used for the input
		// 	instance.Deallocate(inPointer, len(jobBytes))
		// }); err != nil {
		// 	return nil, errors.Wrap(err, "failed to useInstance")
		// }
		//
		// return echo.NewHTTPError(http.StatusNotImplemented)
	}
}
