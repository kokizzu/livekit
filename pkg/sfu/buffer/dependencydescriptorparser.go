package buffer

import (
	"fmt"

	dd "github.com/livekit/livekit-server/pkg/sfu/buffer/dependencydescriptor"

	"github.com/livekit/protocol/logger"
)

type videoLayer struct {
	spatial  int
	temporal int
}

type DependencyDescriptorParser struct {
	structure         *dd.FrameDependencyStructure
	ddExt             uint8
	logger            logger.Logger
	onMaxLayerChanged func(int, int)
	decodeTargetLayer []videoLayer
}

func NewDependencyDescriptorParser(ddExt uint8, logger logger.Logger, onMaxLayerChanged func(int, int)) *DependencyDescriptorParser {
	logger.Infow("creating video stream receiver", "ddExt", ddExt)
	return &DependencyDescriptorParser{
		ddExt:             ddExt,
		logger:            logger,
		onMaxLayerChanged: onMaxLayerChanged,
	}
}

func (r *DependencyDescriptorParser) Parse(ep *ExtPacket) {
	pkt := ep.Packet
	if ddBuf := pkt.GetExtension(r.ddExt); ddBuf != nil {
		var ddVal dd.DependencyDescriptor
		ext := &dd.DependencyDescriptorExtension{
			Descriptor: &ddVal,
			Structure:  r.structure,
		}
		_, err := ext.Unmarshal(ddBuf)
		if err != nil {
			r.logger.Infow("failed to parse generic dependency descriptor", "err", err)
			return
		}

		if ddVal.AttachedStructure != nil && !ddVal.FirstPacketInFrame {
			r.logger.Infow("ignoring non-first packet in frame with attached structure")
			return
		}

		if ddVal.AttachedStructure != nil {
			var maxSpatial, maxTemporal int
			r.structure = ddVal.AttachedStructure
			r.decodeTargetLayer = r.decodeTargetLayer[:0]
			for target := 0; target < r.structure.NumDecodeTargets; target++ {
				layer := videoLayer{0, 0}
				for _, t := range r.structure.Templates {
					if t.DecodeTargetIndications[target] != dd.DecodeTargetNotPresent {
						if layer.spatial < t.SpatialId {
							layer.spatial = t.SpatialId
						}
						if layer.temporal < t.TemporalId {
							layer.temporal = t.TemporalId
						}
					}
				}
				if layer.spatial > maxSpatial {
					maxSpatial = layer.spatial
				}
				if layer.temporal > maxTemporal {
					maxTemporal = layer.temporal
				}
				r.decodeTargetLayer = append(r.decodeTargetLayer, layer)
			}
			r.logger.Debugw("max layer changed", "maxSpatial", maxSpatial, "maxTemporal", maxTemporal)
			r.onMaxLayerChanged(maxSpatial, maxTemporal)
		}

		if ddVal.AttachedStructure != nil && ddVal.FirstPacketInFrame {
			r.logger.Debugw(fmt.Sprintf("parsed dependency descriptor\n%s", ddVal.String()))
		}

		if mask := ddVal.ActiveDecodeTargetsBitmask; mask != nil {
			var maxSpatial, maxTemporal int
			for dt, layer := range r.decodeTargetLayer {
				if *mask&(1<<dt) != uint32(dd.DecodeTargetNotPresent) {
					if maxSpatial < layer.spatial {
						maxSpatial = layer.spatial
					}
					if maxTemporal < layer.temporal {
						maxTemporal = layer.temporal
					}
				}
			}
			r.logger.Debugw("max layer changed", "maxSpatial", maxSpatial, "maxTemporal", maxTemporal)
			r.onMaxLayerChanged(maxSpatial, maxTemporal)
		}
		ep.DependencyDescriptor = &ddVal
		ep.SpatialLayer = int32(ddVal.FrameDependencies.SpatialId)
		ep.TemporalLayer = int32(ddVal.FrameDependencies.TemporalId)
	}
}