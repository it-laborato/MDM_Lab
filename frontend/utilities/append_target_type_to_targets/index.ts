import { INode } from "interfaces/node";
import { map } from "lodash";

export const parseEntityFunc = (node: INode) => {
  let nodeCpuOutput = null;
  if (node) {
    let clockSpeedOutput = null;
    try {
      const clockSpeed =
        node.cpu_brand.split("@ ")[1] || node.cpu_brand.split("@")[1];
      const clockSpeedFlt = parseFloat(clockSpeed.split("GHz")[0].trim());
      clockSpeedOutput = Math.floor(clockSpeedFlt * 10) / 10;
    } catch (e) {
      // Some CPU brand strings do not fit this format and we can't parse the
      // clock speed. Leave it set to 'Unknown'.
      console.log(
        `Unable to parse clock speed from cpu_brand: ${node.cpu_brand}`
      );
    }
    if (node.cpu_physical_cores || clockSpeedOutput) {
      nodeCpuOutput = `${node.cpu_physical_cores || "Unknown"} x ${
        clockSpeedOutput || "Unknown"
      } GHz`;
    }
  }

  const additionalAttrs = {
    cpu_type: nodeCpuOutput,
    target_type: "nodes",
  };

  return {
    ...node,
    ...additionalAttrs,
  };
};

const appendTargetTypeToTargets = (targets: any, targetType: string) => {
  return map(targets, (target) => {
    if (targetType === "nodes") {
      return parseEntityFunc(target);
    }

    return {
      ...target,
      target_type: targetType,
    };
  });
};

export default appendTargetTypeToTargets;
