import React from "react";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";
import { IVulnerabilitiesEmptyStateReason } from "services/entities/vulnerabilities";

export interface IEmptyVulnerabilitiesTableProps {
  isPremiumTier?: boolean;
  teamId?: number;
  exploitedFilter?: boolean;
  isSoftwareDisabled?: boolean;
  emptyStateReason?: IVulnerabilitiesEmptyStateReason;
}


const emptyStateDetails: Record<
  IVulnerabilitiesEmptyStateReason,
  Partial<IEmptyTableProps>
> = {
  "no-vulns-detected": {
    graphicName: "empty-search-question",
    header: "No vulnerabilities detected",
    info: "Expecting to see vulnerabilities? Check back later.",
  },
  "no-matching-items": {
    graphicName: "empty-search-question",
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  },
  "invalid-cve": {
    graphicName: "empty-search-exclamation",
    header: "That vulnerability (CVE) is not valid",
    info:
      'Try updating your search to use CVE format: "CVE-YYYY-<4 or more digits>"',
  },
  "unknown-cve": {
    graphicName: "empty-search-question",
    header: "This is not a known CVE",
    info: "None of Mdmlab's vulnerability sources are aware of this CVE.",
    
  },
  "known-vuln": {
    graphicName: "empty-search-check",
    header:
      "This is a known vulnerability (CVE), but it wasn't detected on any nodes",
   
  },
};

const EmptyVulnerabilitiesTable: React.FC<IEmptyVulnerabilitiesTableProps> = ({
  isPremiumTier,
  teamId,
  exploitedFilter,
  isSoftwareDisabled,
  emptyStateReason,
}) => {
  if (isSoftwareDisabled) {
    return (
      <EmptyTable
        graphicName="empty-search-question"
        header="Software inventory disabled"
        info={
          <>
            Users with the admin role can{" "}.
          </>
        }
      />
    );
  }

  const defaultEmptyState: IEmptyTableProps = {
    graphicName: "empty-search-question",
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  };

  const emptyState = emptyStateReason
    ? { ...defaultEmptyState, ...emptyStateDetails[emptyStateReason] }
    : defaultEmptyState;

  if (emptyStateReason === "known-vuln" && teamId !== undefined) {
    emptyState.header += " in this team";
  }

  if (
    isPremiumTier &&
    exploitedFilter &&
    emptyStateReason !== "unknown-cve" &&
    emptyStateReason !== "invalid-cve"
  ) {
    emptyState.info =
      "Try removing the exploited vulnerabilities filter to expand your search.";
  }

  return <EmptyTable {...emptyState} />;
};

export default EmptyVulnerabilitiesTable;
