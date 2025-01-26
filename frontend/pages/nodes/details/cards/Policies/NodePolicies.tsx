import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import { noop } from "lodash";

import { INodePolicy } from "interfaces/policy";
import { PolicyResponse, SUPPORT_LINK } from "utilities/constants";
import { createNodesByPolicyPath } from "utilities/helpers";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";
import CustomLink from "components/CustomLink";

import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./NodePoliciesTable/NodePoliciesTableConfig";
import PolicyFailingCount from "./NodePoliciesTable/PolicyFailingCount";

const baseClass = "policies-card";

interface IPoliciesProps {
  policies: INodePolicy[];
  isLoading: boolean;
  deviceUser?: boolean;
  togglePolicyDetailsModal: (policy: INodePolicy) => void;
  nodePlatform: string;
  router: InjectedRouter;
  currentTeamId?: number;
}

interface INodePoliciesRowProps extends Row {
  original: {
    id: number;
    response: "pass" | "fail";
  };
}

const Policies = ({
  policies,
  isLoading,
  deviceUser,
  togglePolicyDetailsModal,
  nodePlatform,
  router,
  currentTeamId,
}: IPoliciesProps): JSX.Element => {
  const tableHeaders = generatePolicyTableHeaders(
    togglePolicyDetailsModal,
    currentTeamId
  );
  if (deviceUser) {
    // Remove view all nodes link
    tableHeaders.pop();
  }
  const failingResponses: INodePolicy[] =
    policies.filter((policy: INodePolicy) => policy.response === "fail") || [];

  const onClickRow = useCallback(
    (row: INodePoliciesRowProps) => {
      const { id: policyId, response: policyResponse } = row.original;

      const viewAllNodePath = createNodesByPolicyPath(
        policyId,
        policyResponse === "pass"
          ? PolicyResponse.PASSING
          : PolicyResponse.FAILING,
        currentTeamId
      );

      router.push(viewAllNodePath);
    },
    [router]
  );

  const renderNodePolicies = () => {
    if (nodePlatform === "ios" || nodePlatform === "ipados") {
      return (
        <EmptyTable
          header={<>Policies are not supported for this node</>}
          info={
            <>
              Interested in detecting device health issues on{" "}
              {nodePlatform === "ios" ? "iPhones" : "iPads"}?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    if (policies.length === 0) {
      return (
        <EmptyTable
          header={
            <>
              No policies are checked{" "}
              {deviceUser ? `on your device` : `for this node`}
            </>
          }
          info={
            <>
              Expecting to see policies? Try selecting “Refetch” to ask{" "}
              {deviceUser ? `your device ` : `this node `}
              to report new vitals.
            </>
          }
        />
      );
    }

    return (
      <>
        {failingResponses?.length > 0 && (
          <PolicyFailingCount policyList={policies} deviceUser={deviceUser} />
        )}
        <TableContainer
          columnConfigs={tableHeaders}
          data={generatePolicyDataSet(policies)}
          isLoading={isLoading}
          defaultSortHeader="response"
          defaultSortDirection="asc"
          resultsTitle="policies"
          emptyComponent={() => <></>}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableMultiRowSelect={!deviceUser} // Removes hover/click state if deviceUser
          isClientSidePagination
          onClickRow={deviceUser ? noop : onClickRow}
        />
      </>
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">Policies</p>
      {renderNodePolicies()}
    </Card>
  );
};

export default Policies;
