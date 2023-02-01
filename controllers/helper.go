// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func requeueAfterSeconds(seconds int) (ctrl.Result, error) {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Duration(seconds) * time.Second,
	}, nil
}

func addFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) {
	if !obj.GetDeletionTimestamp().IsZero() {
		return
	}

	if controllerutil.ContainsFinalizer(obj, finalizer) {
		return
	}

	controllerutil.AddFinalizer(obj, finalizer)
	c.Update(ctx, obj)
}
