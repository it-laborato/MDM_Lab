import React, { useCallback, useState, useEffect, useContext } from "react";

import { IPolicyStats } from "interfaces/policy";

import { AppContext } from "context/app";
import { syntaxHighlight } from "utilities/helpers";
import validURL from "components/forms/validators/valid_url";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";

import Slider from "components/forms/fields/Slider";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Icon from "components/Icon";
import CalendarEventPreviewModal from "../CalendarEventPreviewModal";
import CalendarPreview from "../../../../../../assets/images/calendar-preview-720x436@2x.png";

const baseClass = "calendar-events-modal";

interface IFormPolicy {
  name: string;
  id: number;
  isChecked: boolean;
}
export interface ICalendarEventsFormData {
  enabled: boolean;
  url: string;
  policies: IFormPolicy[];
}

interface ICalendarEventsModal {
  onExit: () => void;
  onSubmit: (formData: ICalendarEventsFormData) => void;
  isUpdating: boolean;
  configured: boolean;
  enabled: boolean;
  url: string;
  policies: IPolicyStats[];
}

// allows any policy name to be the name of a form field, one of the checkboxes
type FormNames = string;

const CalendarEventsModal = ({
  onExit,
  onSubmit,
  isUpdating,
  configured,
  enabled,
  url,
  policies,
}: ICalendarEventsModal) => {
  const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);

  const isAdmin = isGlobalAdmin || isTeamAdmin;

  const [formData, setFormData] = useState<ICalendarEventsFormData>({
    enabled,
    url,
    policies: policies.map((policy) => ({
      name: policy.name,
      id: policy.id,
      isChecked: policy.calendar_events_enabled || false,
    })),
  });
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showPreviewCalendarEvent, setShowPreviewCalendarEvent] = useState(
    false
  );
  const [showExamplePayload, setShowExamplePayload] = useState(false);
  const [selectedPolicyToPreview, setSelectedPolicyToPreview] = useState<
    IPolicyStats | undefined
  >();

  // Eliminate race condition after updating policies
  useEffect(() => {
    setFormData({
      ...formData,
      policies: policies.map((policy) => ({
        name: policy.name,
        id: policy.id,
        isChecked: policy.calendar_events_enabled || false,
      })),
    });
  }, [policies]);

  // Used on URL change only when URL error exists and always on attempting to save
  const validateForm = (newFormData: ICalendarEventsFormData) => {
    const errors: Record<string, string> = {};
    const { url: newUrl } = newFormData;
    if (
      formData.enabled &&
      !validURL({ url: newUrl || "", protocols: ["http", "https"] })
    ) {
      const errorPrefix = newUrl ? `${newUrl} is not` : "Please enter";
      errors.url = `${errorPrefix} a valid resolution webhook URL`;
    }

    return errors;
  };

  const onFeatureEnabledChange = () => {
    const newFormData = { ...formData, enabled: !formData.enabled };

    const isDisabling = newFormData.enabled === false;

    // On disabling feature, validate URL and if an error clear input and error
    if (isDisabling) {
      const errors = validateForm(newFormData);

      if (errors.url) {
        newFormData.url = "";
        delete formErrors.url;
        setFormErrors(formErrors);
      }
    }

    setFormData(newFormData);
  };

  const onUrlChange = (value: string) => {
    const newFormData = { ...formData, url: value };
    // On URL change with erroneous URL, validate form
    if (formErrors.url) {
      setFormErrors(validateForm(newFormData));
    }

    setFormData(newFormData);
  };

  const onPolicyEnabledChange = useCallback(
    (newVal: { name: FormNames; value: boolean }) => {
      const { name, value } = newVal;
      const newFormPolicies = formData.policies.map((formPolicy) => {
        if (formPolicy.name === name) {
          return { ...formPolicy, isChecked: value };
        }
        return formPolicy;
      });
      const newFormData = { ...formData, policies: newFormPolicies };
      setFormData(newFormData);
    },
    [formData]
  );

  const onUpdateCalendarEvents = () => {
    const errors = validateForm(formData);

    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
    } else {
      onSubmit(formData);
    }
  };

  const togglePreviewCalendarEvent = () => {
    setShowPreviewCalendarEvent(!showPreviewCalendarEvent);
  };

  const renderExamplePayload = () => {
    return (
      <>
        <pre>POST https://server.com/example</pre>
        <pre
          dangerouslySetInnerHTML={{
            __html: syntaxHighlight({
              timestamp: "0000-00-00T00:00:00Z",
              node_id: 1,
              node_display_name: "Anna's MacBook Pro",
              node_serial_number: "ABCD1234567890",
              failing_policies: [
                {
                  id: 123,
                  name: "macOS - Disable guest account",
                },
              ],
            }),
          }}
        />
      </>
    );
  };

  const renderPolicies = () => {
    return (
      <div className="form-field">
        <div className="form-field__label">Policies:</div>
        <ul className="automated-policies-section">
          {formData.policies.map((policy) => {
            const { isChecked, name, id } = policy;
            return (
              <li className="policy-row" id={`policy-row--${id}`} key={id}>
                <Checkbox
                  value={isChecked}
                  name={name}
                  // can't use parseTarget as value needs to be set to !currentValue
                  onChange={() => {
                    onPolicyEnabledChange({ name, value: !isChecked });
                  }}
                  disabled={!formData.enabled}
                >
                  <TooltipTruncatedText value={name} />
                </Checkbox>
                <Button
                  variant="text-icon"
                  onClick={() => {
                    setSelectedPolicyToPreview(
                      policies.find((p) => p.id === id)
                    );
                    togglePreviewCalendarEvent();
                  }}
                  className="policy-row__preview-button"
                >
                  <Icon name="eye" /> Preview
                </Button>
              </li>
            );
          })}
        </ul>
        <span className="form-field__help-text">
          A calendar event will be created for end users if one of their nodes
          fail any of these policies.{" "}
         
        </span>
      </div>
    );
  };

  const renderPlaceholderModal = () => {
    return (
      <div className="placeholder">
        <a href="https://www.mdmlabdm.com/learn-more-about/calendar-events">
          <img src={CalendarPreview} alt="Calendar preview" />
        </a>
        <div>
          To create calendar events for end users if their nodes fail policies,
          you must first connect Mdmlab to your Google Workspace service account.
        </div>
        <div>
          This can be configured in <b>Settings</b> &gt; <b>Integrations</b>{" "}
          &gt; <b>Calendars.</b>
        </div>
      
        <div className="modal-cta-wrap">
          <Button onClick={onExit} variant="brand">
            Done
          </Button>
        </div>
      </div>
    );
  };

  const renderAdminHeader = () => (
    <div className="form-header">
      <Slider
        value={formData.enabled}
        onChange={onFeatureEnabledChange}
        inactiveText="Disabled"
        activeText="Enabled"
      />
      <Button
        type="button"
        variant="text-link"
        onClick={() => {
          setSelectedPolicyToPreview(undefined);
          togglePreviewCalendarEvent();
        }}
      >
        Preview calendar event
      </Button>
    </div>
  );

  /** Maintainer does not have access to update calendar event
  / configuration only to set the automated policies
  / Modal not available for maintainers if calendar is disabled but
  / disabled inputs here as fail safe and to match admin view.
  */
  const renderMaintainerHeader = () => (
    <>
      <div className="form-header">
        <RevealButton
          isShowing={showExamplePayload}
          className={`${baseClass}__show-example-payload-toggle`}
          hideText="Hide example payload"
          showText="Show example payload"
          caretPosition="after"
          onClick={() => {
            setShowExamplePayload(!showExamplePayload);
          }}
          // disabled={!formData.enabled}
        />
        <Button
          type="button"
          variant="text-link"
          onClick={() => {
            setSelectedPolicyToPreview(undefined);
            togglePreviewCalendarEvent();
          }}
        >
          Preview calendar event
        </Button>
      </div>
      {showExamplePayload && renderExamplePayload()}
    </>
  );

  const renderConfiguredModal = () => (
    <div className={`${baseClass} form`}>
      {isAdmin ? renderAdminHeader() : renderMaintainerHeader()}
      <div
        className={`form ${formData.enabled ? "" : "form-fields--disabled"}`}
      >
        {isAdmin && (
          <>
            <InputField
              placeholder="https://server.com/example"
              label="Resolution webhook URL"
              onChange={onUrlChange}
              name="url"
              value={formData.url}
              error={formErrors.url}
              tooltip="Provide a URL to deliver a webhook request to."
              helpText="A request will be sent to this URL during the calendar event. Use it to trigger auto-remediation."
              disabled={!formData.enabled}
            />
            <RevealButton
              isShowing={showExamplePayload}
              className={`${baseClass}__show-example-payload-toggle`}
              hideText="Hide example payload"
              showText="Show example payload"
              caretPosition="after"
              onClick={() => {
                setShowExamplePayload(!showExamplePayload);
              }}
              disabled={!formData.enabled}
            />
            {showExamplePayload && renderExamplePayload()}
          </>
        )}
        {renderPolicies()}
      </div>
      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onUpdateCalendarEvents}
          className="save-loading"
          isLoading={isUpdating}
          disabled={Object.keys(formErrors).length > 0}
        >
          Save
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );

  if (showPreviewCalendarEvent) {
    return (
      <CalendarEventPreviewModal
        onCancel={togglePreviewCalendarEvent}
        policy={selectedPolicyToPreview}
      />
    );
  }

  return (
    <Modal
      title="Calendar events"
      onExit={onExit}
      onEnter={
        configured
          ? () => {
              onUpdateCalendarEvents();
            }
          : onExit
      }
      className={baseClass}
      width="large"
    >
      {configured ? renderConfiguredModal() : renderPlaceholderModal()}
    </Modal>
  );
};

export default CalendarEventsModal;
