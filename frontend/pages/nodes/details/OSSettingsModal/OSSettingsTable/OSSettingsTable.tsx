import React from "react";
import TableContainer from "components/TableContainer";

import generateTableHeaders, {
  INodeMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "os-settings-table";

interface IOSSettingsTableProps {
  canResendProfiles: boolean;
  nodeId: number;
  tableData: INodeMdmProfileWithAddedStatus[];
  onProfileResent?: () => void;
}

const OSSettingsTable = ({
  canResendProfiles,
  nodeId,
  tableData,
  onProfileResent,
}: IOSSettingsTableProps) => {
  const tableConfig = generateTableHeaders(
    nodeId,
    canResendProfiles,
    onProfileResent
  );

  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columnConfigs={tableConfig}
        data={tableData}
        emptyComponent="symbol"
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
      />
    </div>
  );
};

export default OSSettingsTable;
