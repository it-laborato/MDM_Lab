import React from "react";
import { Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { INodeMdmData } from "interfaces/node";
import {
  MDMLAB_FILEVAULT_PROFILE_DISPLAY_NAME,
  // MDMLAB_FILEVAULT_PROFILE_IDENTIFIER,
  INodeMdmProfile,
  MdmDDMProfileStatus,
  MdmProfileStatus,
  isLinuxDiskEncryptionStatus,
  isWindowsDiskEncryptionStatus,
} from "interfaces/mdm";

import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import OSSettingStatusCell from "./OSSettingStatusCell";
import {
  generateLinuxDiskEncryptionSetting,
  generateWinDiskEncryptionSetting,
} from "../../helpers";
import OSSettingsErrorCell from "./OSSettingsErrorCell";

export const isMdmProfileStatus = (
  status: string
): status is MdmProfileStatus => {
  return status !== "action_required";
};

export interface INodeMdmProfileWithAddedStatus
  extends Omit<INodeMdmProfile, "status"> {
  status: OsSettingsTableStatusValue;
}

type ITableColumnConfig = Column<INodeMdmProfileWithAddedStatus>;
type ITableStringCellProps = IStringCellProps<INodeMdmProfileWithAddedStatus>;

/** Non DDM profiles can have an `action_required` as a profile status.  DDM
 * Profiles will never have this status.
 */
export type INonDDMProfileStatus = MdmProfileStatus | "action_required";

export type OsSettingsTableStatusValue =
  | MdmDDMProfileStatus
  | INonDDMProfileStatus;

const generateTableConfig = (
  nodeId: number,
  canResendProfiles: boolean,
  onProfileResent?: () => void
): ITableColumnConfig[] => {
  return [
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <TooltipTruncatedTextCell
            value={cellProps.cell.value}
            className="os-settings-name-cell"
          />
        );
      },
    },
    {
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <OSSettingStatusCell
            status={cellProps.row.original.status}
            operationType={cellProps.row.original.operation_type}
            profileName={cellProps.row.original.name}
            nodePlatform={cellProps.row.original.platform}
          />
        );
      },
    },
    {
      Header: "Error",
      disableSortBy: true,
      accessor: "detail",
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, platform, status } = cellProps.row.original;
        const isFailedWindowsDiskEncryption =
          platform === "windows" &&
          name === "Disk Encryption" &&
          status === "failed";
        return (
          <OSSettingsErrorCell
            canResendProfiles={
              canResendProfiles && !isFailedWindowsDiskEncryption
            }
            nodeId={nodeId}
            profile={cellProps.row.original}
            onProfileResent={onProfileResent}
          />
        );
      },
    },
  ];
};

const makeWindowsRows = ({ profiles, os_settings }: INodeMdmData) => {
  const rows: INodeMdmProfileWithAddedStatus[] = [];

  if (profiles) {
    rows.push(...profiles);
  }

  if (
    os_settings?.disk_encryption?.status &&
    isWindowsDiskEncryptionStatus(os_settings.disk_encryption.status)
  ) {
    rows.push(
      generateWinDiskEncryptionSetting(
        os_settings.disk_encryption.status,
        os_settings.disk_encryption.detail
      )
    );
  }

  if (rows.length === 0 && !profiles) {
    return null;
  }

  return rows;
};

const makeLinuxRows = ({ profiles, os_settings }: INodeMdmData) => {
  const rows: INodeMdmProfileWithAddedStatus[] = [];

  if (profiles) {
    rows.push(...profiles);
  }

  if (
    os_settings?.disk_encryption?.status &&
    isLinuxDiskEncryptionStatus(os_settings.disk_encryption.status)
  ) {
    rows.push(
      generateLinuxDiskEncryptionSetting(
        os_settings.disk_encryption.status,
        os_settings.disk_encryption.detail
      )
    );
  }

  if (rows.length === 0 && !profiles) {
    return null;
  }

  return rows;
};

const makeDarwinRows = ({ profiles, macos_settings }: INodeMdmData) => {
  if (!profiles) {
    return null;
  }

  let rows: INodeMdmProfileWithAddedStatus[] = profiles;
  if (macos_settings?.disk_encryption === "action_required") {
    rows = profiles.map((p) => {
      // TODO: this is a brittle check for the filevault profile
      // it would be better to match on the identifier but it is not
      // currently available in the API response
      if (p.name === MDMLAB_FILEVAULT_PROFILE_DISPLAY_NAME) {
        return { ...p, status: "action_required" };
      }
      return p;
    });
  }

  return rows;
};

export const generateTableData = (
  nodeMDMData: INodeMdmData,
  platform: string
) => {
  switch (platform) {
    case "windows":
      return makeWindowsRows(nodeMDMData);
    case "darwin":
      return makeDarwinRows(nodeMDMData);
    case "ubuntu":
      return makeLinuxRows(nodeMDMData);
    case "rhel":
      return makeLinuxRows(nodeMDMData);
    case "ios":
      return nodeMDMData.profiles;
    case "ipados":
      return nodeMDMData.profiles;
    default:
      return null;
  }
};

export default generateTableConfig;
