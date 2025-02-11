import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import {
  DEFAULT_TRANSPARENCY_URL,
  IAppConfigFormProps,
  IFormField,
} from "../constants";

interface IMdmlabDesktopFormData {
  transparencyUrl: string;
}
interface IMdmlabDesktopFormErrors {
  transparency_url?: string | null;
}
const baseClass = "app-config-form";

const MdmlabDesktop = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IMdmlabDesktopFormData>({
    transparencyUrl:
      appConfig.mdmlab_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
  });

  const [formErrors, setFormErrors] = useState<IMdmlabDesktopFormErrors>({});

  const onInputChange = ({ value }: IFormField) => {
    setFormData({ transparencyUrl: value.toString() });
    setFormErrors({});
  };

  const validateForm = () => {
    const { transparencyUrl } = formData;

    const errors: IMdmlabDesktopFormErrors = {};
    if (transparencyUrl && !validUrl({ url: transparencyUrl })) {
      errors.transparency_url = `${transparencyUrl} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataForAPI = {
      mdmlab_desktop: {
        transparency_url: formData.transparencyUrl,
      },
    };

    handleSubmit(formDataForAPI);
  };

  if (!isPremiumTier) {
    return <></>;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Mdmlab Desktop" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            When an end user clicks “About Mdmlab” in the Mdmlab Desktop menu, by
            default they are taken to{" "}. You can override the URL to take them to a resource of your
            choice.
          </p>
          <InputField
            label="Custom transparency URL"
            onChange={onInputChange}
            name="transparency_url"
            value={formData.transparencyUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.transparency_url}
            placeholder="https://mdmlabdm.com/transparency"
          />
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
    </div>
  );
};

export default MdmlabDesktop;
