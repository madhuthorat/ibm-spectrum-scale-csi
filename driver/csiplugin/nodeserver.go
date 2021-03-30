/**
 * Copyright 2019 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package scale

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/golang/glog"
	"golang.org/x/net/context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
//	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"
)

type ScaleNodeServer struct {
	Driver *ScaleDriver
	// TODO: Only lock mutually exclusive calls and make locking more fine grained
	mux sync.Mutex
}

func (ns *ScaleNodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodePublishVolume")

	glog.V(4).Infof("NodePublishVolume called with req: %#v", req)

	// Validate Arguments
	targetPath := req.GetTargetPath()
	//sourcePath := req.StagingTargetPath()
	volumeID := req.GetVolumeId()
	volumeCapability := req.GetVolumeCapability()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volumeID must be provided")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target path must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability must be provided")
	}

	if _, ok := volumeCapability.GetAccessType().(*csi.VolumeCapability_Block); ok {
		glog.Infof("***BVS*** For block volume, targetPath: %s", targetPath)

	        splitVId := strings.Split(volumeID, ";")

        	if len(splitVId) < 3 {
                	return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** NodePublishVolume VolumeID is not in proper format, volumeID: %s", volumeID))
	        }

        	index := 2 

	        SlnkPart := splitVId[index]
        	targetSlnkPath := strings.Split(SlnkPart, "=")

	        if len(targetSlnkPath) < 2 {
        	        return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** 1NodePublishVolume VolumeID is not in proper format, volumeID: %s", volumeID))
	        }

        	glog.Infof("***BVS*** Target SpectrumScale Symlink Path : %v\n", targetSlnkPath[1])
		volPath := targetSlnkPath[1]

                // Create loop device using the volume path.
	        args := []string{"-fP", "--show", volPath}
        	loopDevice, err := executeCmd("/usr/sbin/losetup", args)
		loopDevTrim := strings.TrimSpace(string(loopDevice))

	        if err != nil {
        	        glog.Errorf("***BVS*** Create loop device failed for file %v, err: %v", volPath, err)
	                return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to create loop device for path %v: err: %v", volPath, err))
        	}
	        glog.Infof("***BVS*** Created loop device: %s for file: %v", loopDevTrim, volPath)


		/*args := []string{"-j", volPath}
		chkloopDevice, err := executeCmd("/usr/sbin/losetup", args)
                if err != nil {
	 	       glog.Infof("***BVS*** failed to get the loop device with losetup too")
                } else {
                       glog.Infof("***BVS*** got loopdevice: %v", chkloopDevice)
                }*/

                args = []string{"-sf", loopDevTrim, targetPath}
                outputBytes, err := executeCmd("/bin/ln", args)
                glog.Infof("***BVS*** Cmd /bin/ln args: %v Output: %v", args, outputBytes)
                if err != nil {
                        return nil, err
                }

                glog.Errorf("***BVS*** Successfully mounted %s", targetPath)
                return &csi.NodePublishVolumeResponse{}, nil
	}

	glog.Infof("***BVS*** NOT block volume")

	/* <cluster_id>;<filesystem_uuid>;path=<symlink_path> */

	splitVId := strings.Split(volumeID, ";")
	if len(splitVId) < 3 {
		return nil, status.Error(codes.InvalidArgument, "volumeID is not in proper format")
	}

	index := 2
	if len(splitVId) == 4 {
		index = 3
	}

	SlnkPart := splitVId[index]
	targetSlnkPath := strings.Split(SlnkPart, "=")

	if len(targetSlnkPath) < 2 {
		return nil, status.Error(codes.InvalidArgument, "volumeID is not in proper format")
	}

	glog.V(4).Infof("Target SpectrumScale Symlink Path : %v\n", targetSlnkPath[1])
	// Check if Mount Dir/slink exist, if yes delete it
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		glog.V(4).Infof("NodePublishVolume - deleting the targetPath - [%v]", targetPath)
		err := os.Remove(targetPath)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete the target path - [%s]. Error [%v]", targetPath, err.Error()))
		}
	}

	// create symlink
	glog.V(4).Infof("NodePublishVolume - creating symlink [%v] -> [%v]", targetPath, targetSlnkPath[1])
	symlinkerr := os.Symlink(targetSlnkPath[1], targetPath)
	if symlinkerr != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create symlink [%s] -> [%s]. Error [%v]", targetPath, targetSlnkPath[1], symlinkerr.Error()))
	}

	glog.V(4).Infof("successfully mounted %s", targetPath)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *ScaleNodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeUnpublishVolume")
	glog.V(4).Infof("NodeUnpublishVolume called with args: %v", req)
	// Validate Arguments
	targetPath := req.GetTargetPath()
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volumeID must be provided")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target path must be provided")
	}

	glog.V(4).Infof("NodeUnpublishVolume - deleting the targetPath - [%v]", targetPath)
	if err := os.Remove(targetPath); err != nil {
		glog.Infof("***BVS*** Couldn't remove targetPath: %s, err: %v", targetPath, err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove targetPath - [%v]. Error [%v]", targetPath, err.Error()))
	}

	/* For block device delete the loop device */
	if strings.Contains(targetPath, "/volumeDevices/") {
		glog.Infof("***BVS*** About to delete Block Volume Device")
		splitVId := strings.Split(volID, ";")

		if len(splitVId) < 3 {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** NodeUnpublishVolume VolumeID is not in proper format, volumeID: %s", volID))
		}

		index := 2

		SlnkPart := splitVId[index]
		targetSlnkPath := strings.Split(SlnkPart, "=")

		if len(targetSlnkPath) < 2 {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** NodeUnpublishVolume VolumeID is not in proper format, volumeID: %s", volID))
		}

		glog.Infof("***BVS*** Attempting to delete block device at path : %v\n", targetSlnkPath[1])
		volPath := targetSlnkPath[1]

		args := []string{"-j", volPath}
		loopDevice, err := executeCmd("/usr/sbin/losetup", args)
		if err != nil {
			glog.Infof("***BVS*** failed to get the loop device for path: %s err: %v", volPath, err)
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** failed to get the loop device for path: %s err: %v", volPath, err))
		}
		if len(loopDevice) == 0 {
			glog.Infof("***BVS*** failed to get the loop device for path: %s err: %v", volPath, err)
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("***BVS*** failed to get the loop device for path: %s err: %v", volPath, err))
		}
		
		loopDevStr := string(loopDevice)
		loopDevStrSplt := strings.Split(loopDevStr, ":")
		devPath := loopDevStrSplt[0]

		args = []string{"-d", devPath}
		out, err := executeCmd("/usr/sbin/losetup", args)
		if err != nil {
			glog.Infof("***BVS*** failed to delete the loop device: %s, err: %v, out: %v", devPath, err, out)
			return nil, status.Error(codes.Internal, err.Error())
		}
		glog.Infof("***BVS*** Successfully removed the loop device: %s", devPath)	
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *ScaleNodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (
	*csi.NodeStageVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeStageVolume")
	ns.mux.Lock()
	defer ns.mux.Unlock()
	glog.V(4).Infof("NodeStageVolume called with req: %#v", req)

	// Validate Arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	volumeCapability := req.GetVolumeCapability()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

        if _, ok := volumeCapability.GetAccessType().(*csi.VolumeCapability_Block); ok {
                /* For block volume we don't do anything, return from here */
                glog.Infof("***BVS*** Block volume.. we don't do anything here")
        } else {
                glog.Infof("***BVS*** NOT block volume")
        }

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *ScaleNodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (
	*csi.NodeUnstageVolumeResponse, error) {
	glog.V(3).Infof("nodeserver NodeUnstageVolume")
	ns.mux.Lock()
	defer ns.mux.Unlock()
	glog.V(4).Infof("NodeUnstageVolume called with req: %#v", req)

	// Validate arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Staging Target Path must be provided")
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *ScaleNodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	glog.V(4).Infof("NodeGetCapabilities called with req: %#v", req)
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.Driver.nscap,
	}, nil
}

func (ns *ScaleNodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	glog.V(4).Infof("NodeGetInfo called with req: %#v", req)
	return &csi.NodeGetInfoResponse{
		NodeId: ns.Driver.nodeID,
	}, nil
}

func (ns *ScaleNodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
func (ns *ScaleNodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
