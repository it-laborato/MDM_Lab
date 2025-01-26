import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import {
  IMdmlabMaintainedAppFormData,
  IFormValidation,
} from "./MdmlabAppDetailsForm";

type IMessageFunc = (formData: IMdmlabMaintainedAppFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IMdmlabMaintainedAppFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATION_CONFIG: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  preInstallQuery: {
    validations: [
      {
        name: "invalidQuery",
        isValid: (formData) => {
          const query = formData.preInstallQuery;
          return (
            query === undefined || query === "" || validateQuery(query).valid
          );
        },
        message: (formData) => validateQuery(formData.preInstallQuery).error,
      },
    ],
  },
  customTarget: {
    validations: [
      {
        name: "requiredLabelTargets",
        isValid: (formData) => {
          if (formData.targetType === "All nodes") return true;
          // there must be at least one label target selected
          return (
            Object.keys(formData.labelTargets).find(
              (key) => formData.labelTargets[key]
            ) !== undefined
          );
        },
      },
    ],
  },
};

const getErrorMessage = (
  formData: IMdmlabMaintainedAppFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const generateFormValidation = (
  formData: IMdmlabMaintainedAppFormData
) => {
  const formValidation: IFormValidation = {
    isValid: true,
  };

  Object.keys(FORM_VALIDATION_CONFIG).forEach((key) => {
    const objKey = key as IFormValidationKey;
    const failedValidation = FORM_VALIDATION_CONFIG[objKey].validations.find(
      (validation) => !validation.isValid(formData)
    );

    if (!failedValidation) {
      formValidation[objKey] = {
        isValid: true,
      };
    } else {
      formValidation.isValid = false;
      formValidation[objKey] = {
        isValid: false,
        message: getErrorMessage(formData, failedValidation.message),
      };
    }
  });

  return formValidation;
};

export const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAny",
    label: "Include any",
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude any",
    disabled: false,
  },
];

export const generateHelpText = (installType: string, customTarget: string) => {
  if (customTarget === "labelsIncludeAny") {
    return installType === "manual" ? (
      <>
        Software will only be available for install on nodes that{" "}
        <b>have any</b> of these labels:
      </>
    ) : (
      <>
        Software will only be installed on nodes that <b>have any</b> of these
        labels:
      </>
    );
  }

  // this is the case for labelsExcludeAny
  return installType === "manual" ? (
    <>
      Software will only be available for install on nodes that{" "}
      <b>don&apos;t have any</b> of these labels:
    </>
  ) : (
    <>
      Software will only be installed on nodes that <b>don&apos;t have any</b>{" "}
      of these labels:{" "}
    </>
  );
};
