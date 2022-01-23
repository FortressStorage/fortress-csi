package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fortress/csi/util"
)

const (
	// we wait 1m30s for the volume state polling, this leaves 20s for the rest of the function call
	timeoutAttachDetach     = 90 * time.Second
	tickAttachDetach        = 2 * time.Second
	timeoutBackupInitiation = 60 * time.Second
	tickBackupInitiation    = 5 * time.Second
	backupStateCompleted    = "Completed"
)

const (
	// NameMaximumLength restricted the length due to Kubernetes name limitation
	NameMaximumLength = 40
)

type ControllerServer struct {
	nodeID      string
	caps        []*csi.ControllerServiceCapability
	accessModes []*csi.VolumeCapability_AccessMode
}

func NewControllerServer(d *FortressDriver) *ControllerServer {
	return &ControllerServer{
		nodeID: d.nodeID,
		caps: getControllerServiceCapabilities(
			[]csi.ControllerServiceCapability_RPC_Type{
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
				csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
				csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
				csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
			}),
		accessModes: getVolumeCapabilityAccessModes(
			[]csi.VolumeCapability_AccessMode_Mode{
				csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			}),
	}
}

func getVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		logrus.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	return vca
}

func getControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) []*csi.ControllerServiceCapability {
	var cscs []*csi.ControllerServiceCapability

	for _, cap := range cl {
		logrus.Infof("Enabling controller service capability: %v", cap.String())
		cscs = append(cscs, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		})
	}

	return cscs
}

func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	volumeID := util.AutoCorrectName(req.GetName(), NameMaximumLength)
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}

	volumeCaps := req.GetVolumeCapabilities()
	if err := cs.validateVolumeCapabilities(volumeCaps); err != nil {
		return nil, err
	}

	volumeParameters := req.GetParameters()
	if volumeParameters == nil {
		volumeParameters = map[string]string{}
	}

	var reqVolSizeBytes int64
	if req.GetCapacityRange() != nil {
		reqVolSizeBytes = req.GetCapacityRange().GetRequiredBytes()
	}

	if reqVolSizeBytes < util.MinimalVolumeSize {
		logrus.Infof("CreateVolume: volume %s requested capacity %v is smaller than minimal capacity %v, enforcing minimal capacity.", volumeID, reqVolSizeBytes, util.MinimalVolumeSize)
		reqVolSizeBytes = util.MinimalVolumeSize
	}

	// Round up to multiple of 2 * 1024 * 1024
	reqVolSizeBytes = util.RoundUpSize(reqVolSizeBytes)

	volumeSize := fmt.Sprintf("%d", reqVolSizeBytes)
	logrus.Infof("CreateVolume: creating a volume by API client, name: %s, size: %s", volumeID, volumeSize)

	// TODO: create bucket + database record

	volumeSource := req.GetVolumeContentSource()

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: reqVolSizeBytes,
			VolumeContext: volumeParameters,
			ContentSource: volumeSource,
		},
	}, nil
}

func (cs *ControllerServer) validateVolumeCapabilities(volumeCaps []*csi.VolumeCapability) error {
	if volumeCaps == nil {
		return status.Error(codes.InvalidArgument, "Volume Capabilities cannot be empty")
	}

	for _, cap := range volumeCaps {
		if cap.GetMount() == nil && cap.GetBlock() == nil {
			return status.Error(codes.InvalidArgument, "cannot have both mount and block access type be undefined")
		}
		if cap.GetMount() != nil && cap.GetBlock() != nil {
			return status.Error(codes.InvalidArgument, "cannot have both block and mount access type")
		}

		supportedMode := false
		for _, m := range cs.accessModes {
			if cap.GetAccessMode().GetMode() == m.GetMode() {
				supportedMode = true
				break
			}
		}
		if !supportedMode {
			return status.Errorf(codes.InvalidArgument, "access mode %v is not supported", cap.GetAccessMode().Mode.String())
		}
	}

	return nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.caps,
	}, nil
}

func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters(),
		},
	}, nil
}

// ControllerPublishVolume will attach the volume to the specified node
func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Volume ID is missing")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume Node ID is missing")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume VolumeCapability is missing")
	}

	if req.Readonly {
		return nil, status.Error(codes.InvalidArgument, "ControllerPublishVolume read only is not currently supported")
	}

	logrus.WithFields(logrus.Fields{
		"volume-id": req.VolumeId,
		"node-id":   req.NodeId,
	}).Info("Controller Publish Volume: called")

	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume will detach the volume
func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
