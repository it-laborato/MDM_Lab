import React, { useState, useEffect, useMemo } from "react";

import {
  HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
  HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
} from "utilities/constants";

import { getCustomDropdownOptions } from "utilities/helpers";

import NodeStatusWebhookPreviewModal from "pages/admin/components/NodeStatusWebhookPreviewModal";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IGlobalNodeStatusWebhookFormData {
  enableNodeStatusWebhook: boolean;
  destination_url?: string;
  nodeStatusWebhookNodePercentage: number;
  nodeStatusWebhookWindow: number;
}

interface IGlobalNodeStatusWebhookFormErrors {
  destination_url?: string;
}

const baseClass = "app-config-form";

const GlobalNodeStatusWebhook = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [
    showNodeStatusWebhookPreviewModal,
    setShowNodeStatusWebhookPreviewModal,
  ] = useState(false);
  const [formData, setFormData] = useState<IGlobalNodeStatusWebhookFormData>({
    enableNodeStatusWebhook:
      appConfig.webhook_settings.node_status_webhook
        ?.enable_node_status_webhook || false,
    destination_url:
      appConfig.webhook_settings.node_status_webhook?.destination_url || "",
    nodeStatusWebhookNodePercentage:
      appConfig.webhook_settings.node_status_webhook?.node_percentage || 1,
    nodeStatusWebhookWindow:
      appConfig.webhook_settings.node_status_webhook?.days_count || 1,
  });

  const {
    enableNodeStatusWebhook,
    destination_url,
    nodeStatusWebhookNodePercentage,
    nodeStatusWebhookWindow,
  } = formData;

  const [
    formErrors,
    setFormErrors,
  ] = useState<IGlobalNodeStatusWebhookFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IGlobalNodeStatusWebhookFormErrors = {};

    if (enableNodeStatusWebhook) {
      if (!destination_url) {
        errors.destination_url = "Destination URL must be present";
      } else if (!validUrl({ url: destination_url })) {
        errors.destination_url = `${destination_url} is not a valid URL`;
      }
    }

    setFormErrors(errors);
  };

  useEffect(() => {
    validateForm();
  }, [enableNodeStatusWebhook]);

  const toggleNodeStatusWebhookPreviewModal = () => {
    setShowNodeStatusWebhookPreviewModal(!showNodeStatusWebhookPreviewModal);
    return false;
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      webhook_settings: {
        node_status_webhook: {
          enable_node_status_webhook: enableNodeStatusWebhook,
          destination_url,
          node_percentage: nodeStatusWebhookNodePercentage,
          days_count: nodeStatusWebhookWindow,
        },
        failing_policies_webhook:
          appConfig.webhook_settings.failing_policies_webhook,
        vulnerabilities_webhook:
          appConfig.webhook_settings.vulnerabilities_webhook,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const percentageNodesOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
        nodeStatusWebhookNodePercentage,
        (val) => `${val}%`
      ),
    // intentionally omit dependency so options only computed initially
    []
  );

  const windowOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
        nodeStatusWebhookWindow,
        (val) => `${val} day${val !== 1 ? "s" : ""}`
      ),
    // intentionally omit dependency so options only computed initially
    []
  );
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Node status webhook" />
        <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            Send an alert if a portion of your nodes go offline.
          </p>
          <Checkbox
            onChange={onInputChange}
            name="enableNodeStatusWebhook"
            value={enableNodeStatusWebhook}
            parseTarget
          >
            Enable node status webhook
          </Checkbox>
          <p className={`${baseClass}__section-description`}>
            A request will be sent to your configured <b>Destination URL</b> if
            the configured <b>Percentage of nodes</b> have not checked into
            Mdmlab for the configured <b>Number of days</b>.
          </p>
          <Button
            type="button"
            variant="text-link"
            onClick={toggleNodeStatusWebhookPreviewModal}
          >
            Preview request
          </Button>
          {enableNodeStatusWebhook && (
            <>
              <InputField
                placeholder="https://server.com/example"
                label="Destination URL"
                onChange={onInputChange}
                name="destination_url"
                value={destination_url}
                parseTarget
                onBlur={validateForm}
                error={formErrors.destination_url}
                tooltip={
                  <>
                    Provide a URL to deliver <br />
                    the webhook request to.
                  </>
                }
              />
              <Dropdown
                label="Percentage of nodes"
                options={percentageNodesOptions}
                onChange={onInputChange}
                name="nodeStatusWebhookNodePercentage"
                value={nodeStatusWebhookNodePercentage}
                parseTarget
                searchable={false}
                onBlur={validateForm}
                tooltip={
                  <>
                    Select the minimum percentage of nodes that
                    <br />
                    must fail to check into Mdmlab in order to trigger
                    <br />
                    the webhook request.
                  </>
                }
              />
              <Dropdown
                label="Number of days"
                options={windowOptions}
                onChange={onInputChange}
                name="nodeStatusWebhookWindow"
                value={nodeStatusWebhookWindow}
                parseTarget
                searchable={false}
                onBlur={validateForm}
                tooltip={
                  <>
                    Select the minimum number of days that the
                    <br />
                    configured <b>Percentage of nodes</b> must fail to
                    <br />
                    check into Mdmlab in order to trigger the
                    <br />
                    webhook request.
                  </>
                }
              />
            </>
          )}
          <Button
            type="submit"
            variant="brand"
            disabled={Object.keys(formErrors).length > 0}
            className="button-wrap"
            isLoading={isUpdatingSettings}
          >
            Save
          </Button>
        </form>
      </div>
      {showNodeStatusWebhookPreviewModal && (
        <NodeStatusWebhookPreviewModal
          toggleModal={toggleNodeStatusWebhookPreviewModal}
        />
      )}
    </div>
  );
};

export default GlobalNodeStatusWebhook;
