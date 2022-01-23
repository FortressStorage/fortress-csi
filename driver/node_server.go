package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// CryptoKeyProvider specifies how the CryptoKeyValue is retrieved
	// We currently only support passphrase retrieval via direct secret values
	CryptoKeyProvider = "CRYPTO_KEY_PROVIDER"
	CryptoKeyValue    = "CRYPTO_KEY_VALUE"
)

type NodeServer struct {
	nodeID string
	caps   []*csi.NodeServiceCapability
}

func NewNodeServer(d *FortressDriver) *NodeServer {
	return &NodeServer{
		nodeID: d.nodeID,
		caps: getNodeServiceCapabilities(
			[]csi.NodeServiceCapability_RPC_Type{
				csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
				csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			}),
	}
}

// NodePublishVolume will mount the volume /dev/longhorn/<volume_name> to target_path
func (ns *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging Target Path must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target Path must be provided")
	}

	log := logrus.WithFields(logrus.Fields{
		"volume_id":           req.VolumeId,
		"staging_target_path": req.StagingTargetPath,
		"target_path":         req.TargetPath,
	})
	log.Info("Node Publish Volume: called")

	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	mnt := req.VolumeCapability.GetMount()
	for _, flag := range mnt.MountFlags {
		options = append(options, flag)
	}

	//fsType := "ext4"
	//if mnt.FsType != "" {
	//fsType = mnt.FsType
	//}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	targetPath := req.GetStagingTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging path missing in request")
	}

	volumeCapability := req.GetVolumeCapability()
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				//Available: stats.availableBytes,
				//Total:     stats.totalBytes,
				//Used:      stats.usedBytes,
				//Unit:      csi.VolumeUsage_BYTES,
			},
			{
				//Available: stats.availableInodes,
				//Total:     stats.totalInodes,
				//Used:      stats.usedInodes,
				//Unit:      csi.VolumeUsage_INODES,
			},
		},
	}, nil
}

// NodeExpandVolume is designed to expand the file system for ONLINE expansion,
// But Longhorn supports OFFLINE expansion only.
func (ns *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	volumePath := req.GetVolumePath()
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "volume path missing in request")
	}

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}

	return nil, status.Errorf(codes.Unimplemented, "volume %s cannot be expanded driver only supports offline expansion", volumeID)
}

func (ns *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId:            ns.nodeID,
		MaxVolumesPerNode: 0, // technically the scsi kernel limit is the max limit of volumes
	}, nil
}

func (ns *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.caps,
	}, nil
}

func getNodeServiceCapabilities(cs []csi.NodeServiceCapability_RPC_Type) []*csi.NodeServiceCapability {
	var nscs []*csi.NodeServiceCapability

	for _, cap := range cs {
		logrus.Infof("Enabling node service capability: %v", cap.String())
		nscs = append(nscs, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		})
	}

	return nscs
}
