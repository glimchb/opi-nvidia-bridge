// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"errors"
	"fmt"

	"go.einride.tech/aip/fieldbehavior"
	"go.einride.tech/aip/resourceid"
	"go.einride.tech/aip/resourcename"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

func (s *Server) validateCreateNvmeControllerRequest(in *pb.CreateNvmeControllerRequest) error {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return err
	}
	// see https://google.aip.dev/133#user-specified-ids
	if in.NvmeControllerId != "" {
		if err := resourceid.ValidateUserSettable(in.NvmeControllerId); err != nil {
			return err
		}
	}

	if in.NvmeController.Spec.Trtype != pb.NvmeTransportType_NVME_TRANSPORT_PCIE {
		return fmt.Errorf("not supported transport type: %v", in.NvmeController.Spec.Trtype)
	}

	if in.NvmeController.Spec.GetPcieId() == nil {
		return errors.New("invalid endpoint type passed for transport")
	}

	// Validate that a resource name conforms to the restrictions outlined in AIP-122.
	return resourcename.Validate(in.Parent)
}

func (s *Server) validateDeleteNvmeControllerRequest(in *pb.DeleteNvmeControllerRequest) error {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return err
	}
	// Validate that a resource name conforms to the restrictions outlined in AIP-122.
	return resourcename.Validate(in.Name)
}

func (s *Server) validateUpdateNvmeControllerRequest(in *pb.UpdateNvmeControllerRequest) error {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return err
	}
	// Validate that a resource name conforms to the restrictions outlined in AIP-122.
	return resourcename.Validate(in.NvmeController.Name)
}

func (s *Server) validateGetNvmeControllerRequest(in *pb.GetNvmeControllerRequest) error {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return err
	}
	// Validate that a resource name conforms to the restrictions outlined in AIP-122.
	return resourcename.Validate(in.Name)
}

func (s *Server) validateStatsNvmeControllerRequest(in *pb.StatsNvmeControllerRequest) error {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return err
	}
	// Validate that a resource name conforms to the restrictions outlined in AIP-122.
	return resourcename.Validate(in.Name)
}
