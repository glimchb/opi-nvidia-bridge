// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"
	"path"
	"sort"

	"github.com/opiproject/gospdk/spdk"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-nvidia-bridge/pkg/models"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"

	"github.com/google/uuid"
	"go.einride.tech/aip/fieldbehavior"
	"go.einride.tech/aip/fieldmask"
	"go.einride.tech/aip/resourceid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func sortNvmeSubsystems(subsystems []*pb.NvmeSubsystem) {
	sort.Slice(subsystems, func(i int, j int) bool {
		return subsystems[i].Spec.Nqn < subsystems[j].Spec.Nqn
	})
}

// CreateNvmeSubsystem creates an Nvme Subsystem
func (s *Server) CreateNvmeSubsystem(ctx context.Context, in *pb.CreateNvmeSubsystemRequest) (*pb.NvmeSubsystem, error) {
	// check input correctness
	if err := s.validateCreateNvmeSubsystemRequest(in); err != nil {
		return nil, err
	}
	// see https://google.aip.dev/133#user-specified-ids
	resourceID := resourceid.NewSystemGenerated()
	if in.NvmeSubsystemId != "" {
		log.Printf("client provided the ID of a resource %v, ignoring the name field %v", in.NvmeSubsystemId, in.NvmeSubsystem.Name)
		resourceID = in.NvmeSubsystemId
	}
	in.NvmeSubsystem.Name = utils.ResourceIDToSubsystemName(resourceID)
	// idempotent API when called with same key, should return same object
	subsys, ok := s.Subsystems[in.NvmeSubsystem.Name]
	if ok {
		log.Printf("Already existing NvmeSubsystem with id %v", in.NvmeSubsystem.Name)
		return subsys, nil
	}
	// check if another object exists with same NQN, it is not allowed
	for _, item := range s.Subsystems {
		if in.NvmeSubsystem.Spec.Nqn == item.Spec.Nqn {
			msg := fmt.Sprintf("Could not create NQN: %s since object %s with same NQN already exists", in.NvmeSubsystem.Spec.Nqn, item.Name)
			return nil, status.Errorf(codes.AlreadyExists, msg)
		}
	}
	// not found, so create a new one
	params := models.NvdaSubsystemNvmeCreateParams{
		Nqn:          in.NvmeSubsystem.Spec.Nqn,
		SerialNumber: in.NvmeSubsystem.Spec.SerialNumber,
		ModelNumber:  in.NvmeSubsystem.Spec.ModelNumber,
	}
	var result models.NvdaSubsystemNvmeCreateResult
	err := s.rpc.Call(ctx, "subsystem_nvme_create", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not create NQN: %s", in.NvmeSubsystem.Spec.Nqn)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	var ver spdk.GetVersionResult
	err = s.rpc.Call(ctx, "spdk_get_version", nil, &ver)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", ver)
	response := utils.ProtoClone(in.NvmeSubsystem)
	response.Status = &pb.NvmeSubsystemStatus{FirmwareRevision: ver.Version}
	s.Subsystems[in.NvmeSubsystem.Name] = response
	return response, nil
}

// DeleteNvmeSubsystem deletes an Nvme Subsystem
func (s *Server) DeleteNvmeSubsystem(ctx context.Context, in *pb.DeleteNvmeSubsystemRequest) (*emptypb.Empty, error) {
	// check input correctness
	if err := s.validateDeleteNvmeSubsystemRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	subsys, ok := s.Subsystems[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		return nil, err
	}
	params := models.NvdaSubsystemNvmeDeleteParams{
		Nqn: subsys.Spec.Nqn,
	}
	var result models.NvdaSubsystemNvmeDeleteResult
	err := s.rpc.Call(ctx, "subsystem_nvme_delete", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete NQN: %s", subsys.Spec.Nqn)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	delete(s.Subsystems, subsys.Name)
	return &emptypb.Empty{}, nil
}

// UpdateNvmeSubsystem updates an Nvme Subsystem
func (s *Server) UpdateNvmeSubsystem(_ context.Context, in *pb.UpdateNvmeSubsystemRequest) (*pb.NvmeSubsystem, error) {
	// check input correctness
	if err := s.validateUpdateNvmeSubsystemRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	volume, ok := s.Subsystems[in.NvmeSubsystem.Name]
	if !ok {
		if in.AllowMissing {
			log.Printf("TODO: in case of AllowMissing, create a new resource, don;t return error")
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.NvmeSubsystem.Name)
		return nil, err
	}
	resourceID := path.Base(volume.Name)
	// update_mask = 2
	if err := fieldmask.Validate(in.UpdateMask, in.NvmeSubsystem); err != nil {
		return nil, err
	}
	log.Printf("TODO: use resourceID=%v", resourceID)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNvmeSubsystem method is not implemented")
}

// ListNvmeSubsystems lists Nvme Subsystems
func (s *Server) ListNvmeSubsystems(ctx context.Context, in *pb.ListNvmeSubsystemsRequest) (*pb.ListNvmeSubsystemsResponse, error) {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	size, offset, perr := utils.ExtractPagination(in.PageSize, in.PageToken, s.Pagination)
	if perr != nil {
		return nil, perr
	}
	var result []models.NvdaSubsystemNvmeListResult
	err := s.rpc.Call(ctx, "subsystem_nvme_list", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	token, hasMoreElements := "", false
	log.Printf("Limiting result len(%d) to [%d:%d]", len(result), offset, size)
	result, hasMoreElements = utils.LimitPagination(result, offset, size)
	if hasMoreElements {
		token = uuid.New().String()
		s.Pagination[token] = offset + size
	}
	Blobarray := make([]*pb.NvmeSubsystem, len(result))
	for i := range result {
		r := &result[i]
		Blobarray[i] = &pb.NvmeSubsystem{Spec: &pb.NvmeSubsystemSpec{Nqn: r.Nqn, SerialNumber: r.SerialNumber, ModelNumber: r.ModelNumber}}
	}
	sortNvmeSubsystems(Blobarray)
	return &pb.ListNvmeSubsystemsResponse{NvmeSubsystems: Blobarray}, nil
}

// GetNvmeSubsystem gets Nvme Subsystems
func (s *Server) GetNvmeSubsystem(ctx context.Context, in *pb.GetNvmeSubsystemRequest) (*pb.NvmeSubsystem, error) {
	// check input correctness
	if err := s.validateGetNvmeSubsystemRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	subsys, ok := s.Subsystems[in.Name]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		return nil, err
	}
	var result []models.NvdaSubsystemNvmeListResult
	err := s.rpc.Call(ctx, "subsystem_nvme_list", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result {
		r := &result[i]
		if r.Nqn == subsys.Spec.Nqn {
			return &pb.NvmeSubsystem{Spec: &pb.NvmeSubsystemSpec{Nqn: r.Nqn, SerialNumber: r.SerialNumber, ModelNumber: r.ModelNumber}, Status: &pb.NvmeSubsystemStatus{FirmwareRevision: "TBD"}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NQN: %s", subsys.Spec.Nqn)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// StatsNvmeSubsystem gets Nvme Subsystem stats
func (s *Server) StatsNvmeSubsystem(_ context.Context, in *pb.StatsNvmeSubsystemRequest) (*pb.StatsNvmeSubsystemResponse, error) {
	// check input correctness
	if err := s.validateStatsNvmeSubsystemRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	return nil, status.Errorf(codes.Unimplemented, "UpdateNvmeSubsystem method is not implemented")
}
