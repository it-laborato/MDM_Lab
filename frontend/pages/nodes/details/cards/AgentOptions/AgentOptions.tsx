import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";
import React from "react";

import { secondsToHms } from "utilities/helpers";

import DataSet from "components/DataSet";
import Card from "components/Card";

const baseClass = "agent-options-card";
interface IAgentOptionsProps {
  osqueryData: { [key: string]: any };
  wrapMdmlabHelper: (helperFn: (value: any) => string, value: string) => string;
  isChromeOS?: boolean;
}

const CHROMEOS_AGENT_OPTIONS = ["Not supported", "Not supported", "10 secs"];
const CHROMEOS_AGENT_OPTIONS_TOOLTIP_MESSAGE =
  "Chromebooks ignore Mdmlab’s agent options configuration. The options displayed below are the same for all Chromebooks.";
const AgentOptions = ({
  osqueryData,
  wrapMdmlabHelper,
  isChromeOS = false,
}: IAgentOptionsProps): JSX.Element => {
  const classNames = classnames(baseClass, {
    [`${baseClass}__chrome-os`]: isChromeOS,
  });

  let configTLSRefresh;
  let loggerTLSPeriod;
  let distributedInterval;

  if (isChromeOS) {
    [
      configTLSRefresh,
      loggerTLSPeriod,
      distributedInterval,
    ] = CHROMEOS_AGENT_OPTIONS;
  } else {
    configTLSRefresh = wrapMdmlabHelper(
      secondsToHms,
      osqueryData.config_tls_refresh
    );
    loggerTLSPeriod = wrapMdmlabHelper(
      secondsToHms,
      osqueryData.logger_tls_period
    );
    distributedInterval = wrapMdmlabHelper(
      secondsToHms,
      osqueryData.distributed_interval
    );
  }

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={classNames}
    >
      {isChromeOS ? (
        <TooltipWrapper
          tipContent={CHROMEOS_AGENT_OPTIONS_TOOLTIP_MESSAGE}
          className="card__header"
        >
          Agent options
        </TooltipWrapper>
      ) : (
        <p className="card__header">Agent options</p>
      )}
      <div className={`${baseClass}__data`}>
        <DataSet title="Config TLS refresh" value={configTLSRefresh} />
        <DataSet title="Logger TLS period" value={loggerTLSPeriod} />
        <DataSet title="Distributed interval" value={distributedInterval} />
      </div>
    </Card>
  );
};

export default AgentOptions;
