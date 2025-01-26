import React, { useCallback, useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";

import useTeamIdParam from "hooks/useTeamIdParam";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
  HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
} from "utilities/constants";

import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import { IDropdownOption } from "interfaces/dropdownOption";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import { getCustomDropdownOptions } from "utilities/helpers";

import NodeStatusWebhookPreviewModal from "pages/admin/components/NodeStatusWebhookPreviewModal";

import validURL from "components/forms/validators/valid_url";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Checkbox from "components/forms/fields/Checkbox";

import TeamNodeExpiryToggle from "./components/TeamNodeExpiryToggle";

const baseClass = "team-settings";

type ITeamSettingsFormData = {
  teamNodeExpiryEnabled: boolean;
  teamNodeExpiryWindow: number | string;
  teamNodeStatusWebhookEnabled: boolean;
  teamNodeStatusWebhookDestinationUrl: string;
  teamNodeStatusWebhookNodePercentage: number;
  teamNodeStatusWebhookWindow: number;
};

type FormNames = keyof ITeamSettingsFormData;

const HOST_EXPIRY_ERROR_TEXT = "Node expiry window must be a positive number.";

const validateTeamSettingsFormData = (
  // will never be called if global setting is not loaded, default to satisfy typechecking
  curGlobalNodeExpiryEnabled = false,
  curFormData: ITeamSettingsFormData
) => {
  const errors: Record<string, string> = {};

  // validate node expiry fields
  const numNodeExpiryWindow = Number(curFormData.teamNodeExpiryWindow);
  if (
    // with no global setting, team window can't be empty if enabled
    (!curGlobalNodeExpiryEnabled &&
      curFormData.teamNodeExpiryEnabled &&
      !numNodeExpiryWindow) ||
    // if nonempty, must be a positive number
    isNaN(numNodeExpiryWindow) ||
    // if overriding a global setting, can be empty to disable local setting
    numNodeExpiryWindow < 0
  ) {
    errors.node_expiry_window = HOST_EXPIRY_ERROR_TEXT;
  }

  // validate node webhook fields
  if (curFormData.teamNodeStatusWebhookEnabled) {
    if (!validURL({ url: curFormData.teamNodeStatusWebhookDestinationUrl })) {
      const errorPrefix = curFormData.teamNodeStatusWebhookDestinationUrl
        ? `${curFormData.teamNodeStatusWebhookDestinationUrl} is not`
        : "Please enter";
      errors.node_status_webhook_destination_url = `${errorPrefix} a valid webhook destination URL`;
    }
  }

  return errors;
};

const TeamSettings = ({ location, router }: ITeamSubnavProps) => {
  const [formData, setFormData] = useState<ITeamSettingsFormData>({
    teamNodeExpiryEnabled: false,
    teamNodeExpiryWindow: "" as number | string,
    teamNodeStatusWebhookEnabled: false,
    teamNodeStatusWebhookDestinationUrl: "",
    teamNodeStatusWebhookNodePercentage: 1,
    teamNodeStatusWebhookWindow: 1,
  });
  // stateful approach required since initial options come from team config api response
  const [isInitialTeamConfig, setIsInitialTeamConfig] = useState(true);
  const [
    percentageNodesDropdownOptions,
    setPercentageNodesDropdownOptions,
  ] = useState<IDropdownOption[]>([]);
  const [windowDropdownOptions, setWindowDropdownOptions] = useState<
    IDropdownOption[]
  >([]);
  const [updatingTeamSettings, setUpdatingTeamSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [
    showNodeStatusWebhookPreviewModal,
    setShowNodeStatusWebhookPreviewModal,
  ] = useState(false);

  const toggleNodeStatusWebhookPreviewModal = () => {
    setShowNodeStatusWebhookPreviewModal(!showNodeStatusWebhookPreviewModal);
  };

  const { renderFlash } = useContext(NotificationContext);

  const { isRouteOk, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    error: errorLoadGlobalConfig,
  } = useQuery<IConfig, Error, IConfig>(
    ["globalConfig"],
    () => configAPI.loadAll(),
    { refetchOnWindowFocus: false }
  );
  const {
    node_expiry_settings: {
      node_expiry_enabled: globalNodeExpiryEnabled,
      node_expiry_window: globalNodeExpiryWindow,
    },
  } = appConfig ?? { node_expiry_settings: {} };

  const {
    data: teamConfig,
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
    error: errorLoadTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teamConfig", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (tC) => {
        setFormData({
          // node expiry settings
          teamNodeExpiryEnabled:
            tC?.node_expiry_settings?.node_expiry_enabled ?? false,
          teamNodeExpiryWindow:
            tC?.node_expiry_settings?.node_expiry_window ?? "",
          // node status webhook settings
          teamNodeStatusWebhookEnabled:
            tC?.webhook_settings?.node_status_webhook
              ?.enable_node_status_webhook ?? false,
          teamNodeStatusWebhookDestinationUrl:
            tC?.webhook_settings?.node_status_webhook?.destination_url ?? "",
          teamNodeStatusWebhookNodePercentage:
            tC?.webhook_settings?.node_status_webhook?.node_percentage ?? 1,
          teamNodeStatusWebhookWindow:
            tC?.webhook_settings?.node_status_webhook?.days_count ?? 1,
        });
      },
    }
  );

  useEffect(() => {
    if (isInitialTeamConfig) {
      setPercentageNodesDropdownOptions(
        getCustomDropdownOptions(
          HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
          teamConfig?.webhook_settings?.node_status_webhook?.node_percentage ??
            1,
          (val) => `${val}%`
        )
      );

      setWindowDropdownOptions(
        getCustomDropdownOptions(
          HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
          teamConfig?.webhook_settings?.node_status_webhook?.days_count ?? 1,
          (val) => `${val} day${val !== 1 ? "s" : ""}`
        )
      );
    }
    // no need for isInitialTeamConfig dependence, since this effect should only run on initial
    // config load
  }, [teamConfig]);

  const onInputChange = useCallback(
    (newVal: { name: FormNames; value: string | number | boolean }) => {
      const { name, value } = newVal;
      const newFormData = { ...formData, [name]: value };
      setFormData(newFormData);
      setFormErrors(
        validateTeamSettingsFormData(globalNodeExpiryEnabled, newFormData)
      );
    },
    [formData, globalNodeExpiryEnabled]
  );

  const updateTeamSettings = useCallback(
    (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();

      setUpdatingTeamSettings(true);
      const castedNodeExpiryWindow = Number(formData.teamNodeExpiryWindow);
      let enableNodeExpiry;
      if (globalNodeExpiryEnabled) {
        if (!castedNodeExpiryWindow) {
          enableNodeExpiry = false;
        } else {
          enableNodeExpiry = formData.teamNodeExpiryEnabled;
        }
      } else {
        enableNodeExpiry = formData.teamNodeExpiryEnabled;
      }
      teamsAPI
        .update(
          {
            node_expiry_settings: {
              node_expiry_enabled: enableNodeExpiry,
              node_expiry_window: castedNodeExpiryWindow,
            },
            webhook_settings: {
              node_status_webhook: {
                enable_node_status_webhook:
                  formData.teamNodeStatusWebhookEnabled,
                destination_url: formData.teamNodeStatusWebhookDestinationUrl,
                node_percentage: formData.teamNodeStatusWebhookNodePercentage,
                days_count: formData.teamNodeStatusWebhookWindow,
              },
            },
          },
          teamIdForApi
        )
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
          refetchTeamConfig();
          setIsInitialTeamConfig(false);
        })
        .catch((errorResponse: { data: IApiError }) => {
          renderFlash(
            "error",
            `Could not update team settings. ${errorResponse.data.errors[0].reason}`
          );
        })
        .finally(() => {
          setUpdatingTeamSettings(false);
        });
    },
    [
      formData,
      globalNodeExpiryEnabled,
      refetchTeamConfig,
      renderFlash,
      teamIdForApi,
    ]
  );

  const renderForm = () => {
    if (errorLoadGlobalConfig || errorLoadTeamConfig) {
      return <DataError />;
    }
    if (isLoadingTeamConfig || isLoadingAppConfig) {
      return <Spinner />;
    }
    return (
      <form onSubmit={updateTeamSettings}>
        <SectionHeader title="Webhook settings" />
        <Checkbox
          name="teamNodeStatusWebhookEnabled"
          onChange={onInputChange}
          parseTarget
          value={formData.teamNodeStatusWebhookEnabled}
          helpText="This will trigger webhooks specific to this team, separate from the global node status webhook."
          tooltipContent="Send an alert if a portion of your nodes go offline."
        >
          Enable node status webhook
        </Checkbox>
        <Button
          type="button"
          variant="text-link"
          onClick={toggleNodeStatusWebhookPreviewModal}
        >
          Preview request
        </Button>
        {formData.teamNodeStatusWebhookEnabled && (
          <>
            <InputField
              placeholder="https://server.com/example"
              label="Node status webhook destination URL"
              onChange={onInputChange}
              name="teamNodeStatusWebhookDestinationUrl"
              value={formData.teamNodeStatusWebhookDestinationUrl}
              parseTarget
              error={formErrors.node_status_webhook_destination_url}
              tooltip={
                <p>
                  Provide a URL to deliver <br />
                  the webhook request to.
                </p>
              }
            />
            <Dropdown
              label="Node status webhook %"
              options={percentageNodesDropdownOptions}
              onChange={onInputChange}
              name="teamNodeStatusWebhookNodePercentage"
              value={formData.teamNodeStatusWebhookNodePercentage}
              parseTarget
              searchable={false}
              tooltip={
                <p>
                  Select the minimum percentage of nodes that
                  <br />
                  must fail to check into Mdmlab in order to trigger
                  <br />
                  the webhook request.
                </p>
              }
            />
            <Dropdown
              label="Node status webhook window"
              options={windowDropdownOptions}
              onChange={onInputChange}
              name="teamNodeStatusWebhookWindow"
              value={formData.teamNodeStatusWebhookWindow}
              parseTarget
              searchable={false}
              tooltip={
                <p>
                  Select the minimum number of days that the
                  <br />
                  configured <b>Percentage of nodes</b> must fail to
                  <br />
                  check into Mdmlab in order to trigger the
                  <br />
                  webhook request.
                </p>
              }
            />
          </>
        )}
        <SectionHeader title="Node expiry settings" />
        {globalNodeExpiryEnabled !== undefined && (
          <TeamNodeExpiryToggle
            globalNodeExpiryEnabled={globalNodeExpiryEnabled}
            globalNodeExpiryWindow={globalNodeExpiryWindow}
            teamExpiryEnabled={formData.teamNodeExpiryEnabled}
            setTeamExpiryEnabled={(isEnabled: boolean) =>
              onInputChange({ name: "teamNodeExpiryEnabled", value: isEnabled })
            }
          />
        )}
        {formData.teamNodeExpiryEnabled && (
          <InputField
            label="Node expiry window"
            // type="text" allows `validate` to differentiate between
            // non-numerical input and an empty input
            type="text"
            onChange={onInputChange}
            parseTarget
            name="teamNodeExpiryWindow"
            value={formData.teamNodeExpiryWindow}
            error={formErrors.node_expiry_window}
          />
        )}
        <Button
          type="submit"
          variant="brand"
          className="button-wrap"
          isLoading={updatingTeamSettings}
          disabled={Object.keys(formErrors).length > 0}
        >
          Save
        </Button>
      </form>
    );
  };

  return (
    <section className={`${baseClass}`}>
      {renderForm()}
      {showNodeStatusWebhookPreviewModal && (
        <NodeStatusWebhookPreviewModal
          toggleModal={toggleNodeStatusWebhookPreviewModal}
          isTeamScope
        />
      )}
    </section>
  );
};
export default TeamSettings;
