package main

import (
	"time"

	"github.com/pborman/uuid"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

func init() {
	workflow.Register(SampleImageFinderWorkflow)
	workflow.Register(SampleImageProcessingWorkflow)
}

// SampleImageFinderWorkflow is a perpetual workflow.
//
// We don't have a reason to stop it, it must be running at all times. It
// returns ContinueAsNewError to ensure that a new instance is always created.
func SampleImageFinderWorkflow(ctx workflow.Context) error {
	activityOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Minute * 10,
			ExpirationInterval:       time.Minute * 10,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"non-retryable error"},
		},
	})
	var imageAddress string
	err := workflow.ExecuteActivity(activityOpts, findRandomImageActivityName).Get(activityOpts, &imageAddress)
	if err != nil {
		return workflow.NewContinueAsNewError(ctx, SampleImageFinderWorkflow)
	}

	// We've found an image! Let's start a child workflow to process it.
	//
	// TODO: figure out what's the impact of the retry policy that we're setting in this child workflow?
	//
	childWorkflowOpts := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:                   "image_processing_" + uuid.New(),
		ExecutionStartToCloseTimeout: time.Hour,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Minute * 10,
			ExpirationInterval:       time.Minute * 10,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"non-retryable error"},
		},
	})
	var childExecution workflow.Execution
	var recreateToken = []byte("")
	if err := workflow.ExecuteChildWorkflow(childWorkflowOpts, SampleImageProcessingWorkflow, imageAddress, recreateToken).GetChildWorkflowExecution().Get(childWorkflowOpts, &childExecution); err != nil {
		return err
	}

	return workflow.NewContinueAsNewError(ctx, SampleImageFinderWorkflow)
}

func SampleImageProcessingWorkflow(ctx workflow.Context, imageAddress string, recreateToken []byte) error {
	var (
		activityOpts = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: time.Minute,
			StartToCloseTimeout:    time.Minute,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          time.Second,
				BackoffCoefficient:       2,
				MaximumInterval:          time.Minute * 10,
				ExpirationInterval:       time.Minute * 10,
				MaximumAttempts:          5,
				NonRetriableErrorReasons: []string{"non-retryable error"},
			},
		})
		sessOpts = &workflow.SessionOptions{
			ExecutionTimeout: time.Minute,
			CreationTimeout:  time.Minute,
		}

		sessCtx workflow.Context
		err     error
	)

	if len(recreateToken) > 0 {
		sessCtx, err = workflow.RecreateSession(activityOpts, recreateToken, sessOpts)
	} else {
		sessCtx, err = workflow.CreateSession(activityOpts, sessOpts)
	}
	if err != nil {
		return err
	}
	defer workflow.CompleteSession(sessCtx)

	var path string
	err = workflow.ExecuteActivity(sessCtx, downloadImageActivity, imageAddress).Get(sessCtx, &path)
	if err != nil {
		return err
		// return workflow.NewContinueAsNewError(
		// 	ctx, SampleImageProcessingWorkflow,
		// 	imageAddress, workflow.GetSessionInfo(sessCtx).GetRecreateToken(),
		// )
	}

	var checksum string
	err = workflow.ExecuteActivity(sessCtx, calcChecksumActivity, path).Get(sessCtx, &checksum)
	if err != nil {
		return err
		// return workflow.NewContinueAsNewError(
		// 	ctx, SampleImageProcessingWorkflow,
		// 	imageAddress, workflow.GetSessionInfo(sessCtx).GetRecreateToken(),
		// )
	}

	workflow.GetLogger(ctx).Info("Image processing workflow completed!", zap.String("checksum", checksum))

	return nil
}
