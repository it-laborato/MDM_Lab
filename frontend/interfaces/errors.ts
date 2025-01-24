import { AxiosError, isAxiosError } from "axios";

/**
 * IMdmlabApiError is the shape of a Mdmlab API error. It represents an element of the `errors`
 * array in a Mdmlab API response for failed requests (see `IMdmlabApiResponseWithErrors`).
 */
export interface IMdmlabApiError {
  name: string;
  reason: string;
}

/**
 * IApiError is the shape of a Mdmlab API response for failed requests.
 *
 * TODO: Rename to IMdmlabApiResponseWithErrors
 */
export interface IApiError {
  message: string;
  errors: IMdmlabApiError[];
  uuid?: string;
}

const isMdmlabApiError = (err: unknown): err is IMdmlabApiError => {
  if (!err || typeof err !== "object" || !("name" in err && "reason" in err)) {
    return false;
  }
  const e = err as Record<"name" | "reason", unknown>;
  if (typeof e.name !== "string" || typeof e.reason !== "string") {
    return false;
  }
  return true;
};

interface IRecordWithErrors extends Record<string | number | symbol, unknown> {
  errors: unknown[];
}

const isRecordWithErrors = (r: unknown): r is IRecordWithErrors => {
  if (!r || typeof r !== "object" || !("errors" in r)) {
    return false;
  }
  const { errors } = r as { errors: unknown };
  if (!Array.isArray(errors)) {
    return false;
  }
  return true;
};

interface IRecordWithDataErrors
  extends Record<string | number | symbol, unknown> {
  data: IRecordWithErrors;
}

const isRecordWithDataErrors = (r: unknown): r is IRecordWithDataErrors => {
  if (!r || typeof r !== "object" || !("data" in r)) {
    return false;
  }
  const { data } = r as { data: unknown };
  if (!isRecordWithErrors(data)) {
    return false;
  }
  const { errors } = data;
  if (!Array.isArray(errors)) {
    return false;
  }
  return true;
};

interface IRecordWithResponseDataErrors
  extends Record<string | number | symbol, unknown> {
  response: IRecordWithDataErrors;
}

const isRecordWithResponseDataErrors = (
  r: unknown
): r is IRecordWithResponseDataErrors => {
  if (!r || typeof r !== "object" || !("response" in r)) {
    return false;
  }
  const { response } = r as { response: unknown };
  if (!isRecordWithDataErrors(response)) {
    return false;
  }
  return true;
};

interface IFilterMdmlabErrorBase {
  nameEquals?: string;
  reasonIncludes?: string;
}

interface IFilterMdmlabErrorName extends IFilterMdmlabErrorBase {
  nameEquals: string;
  reasonIncludes?: never;
}

interface IFilterMdmlabErrorReason extends IFilterMdmlabErrorBase {
  nameEquals?: never;
  reasonIncludes: string;
}

// FilterMdmlabError is the shape of a filter that can be applied to to filter Mdmlab
// server errors. It is the union of FilterMdmlabErrorName and FilterMdmlabErrorReason,
// which ensures that only one of `nameEquals` or `reasonIncludes` can be specified.
type IFilterMdmlabError = IFilterMdmlabErrorName | IFilterMdmlabErrorReason;

const filterMdmlabErrorNameEquals = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => isMdmlabApiError(e) && e.name === value) as
    | IMdmlabApiError
    | undefined;
};

const filterMdmlabErrorReasonIncludes = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => isMdmlabApiError(e) && e.reason?.includes(value)) as
    | IMdmlabApiError
    | undefined;
};

const getReasonFromErrors = (errors: unknown[], filter?: IFilterMdmlabError) => {
  if (!errors.length) {
    return "";
  }

  let mdmlabError: IMdmlabApiError | undefined;
  if (filter?.nameEquals) {
    mdmlabError = filterMdmlabErrorNameEquals(errors, filter.nameEquals);
  } else if (filter?.reasonIncludes) {
    mdmlabError = filterMdmlabErrorReasonIncludes(errors, filter.reasonIncludes);
  } else {
    mdmlabError = isMdmlabApiError(errors[0]) ? errors[0] : undefined;
  }

  return mdmlabError?.reason || "";
};

const getReasonFromRecordWithDataErrors = (
  r: IRecordWithDataErrors,
  filter?: IFilterMdmlabError
): string => {
  return getReasonFromErrors(r.data.errors, filter);
};

const getReasonFromAxiosError = (
  ae: AxiosError,
  filter?: IFilterMdmlabError
): string => {
  return isRecordWithDataErrors(ae.response)
    ? getReasonFromRecordWithDataErrors(ae.response, filter)
    : "";
};

/**
 * getErrorReason attempts to parse a unknown payload as an `AxiosError` or
 * other `Record`-like object with the general shape as follows:
 * `{ response: { data: { errors: unknown[] } } }`
 *
 * It attempts to extract a `reason` from a Mdmlab API error (i.e. an object
 * with `name` and `reason` properties) in the `errors` array, if present.
 * Other in values in the payload are generally ignored.
 *
 * If `filter` is specified, it attempts to find an error that satisfies the filter
 * and returns the `reason`, if found. Otherwise, it returns the `reason`
 * of the first error, if any.
 *
 * By default, an empty string is returned as the reason if no error is found.
 */
export const getErrorReason = (
  payload: unknown | undefined,
  filter?: IFilterMdmlabError
): string => {
  if (isAxiosError(payload)) {
    return getReasonFromAxiosError(payload, filter);
  }

  if (isRecordWithResponseDataErrors(payload)) {
    return getReasonFromRecordWithDataErrors(payload.response, filter);
  }

  if (isRecordWithDataErrors(payload)) {
    return getReasonFromRecordWithDataErrors(payload, filter);
  }

  if (isRecordWithErrors(payload)) {
    return getReasonFromErrors(payload.errors, filter);
  }

  return "";
};

export const ignoreAxiosError = (err: AxiosError, ignoreStatuses: number[]) => {
  // TODO - isAxiosError currently not recognizing axios error, fix
  // if (!isAxiosError(err)) {
  //   return false;
  // }
  // return !!err.response && ignoreStatuses.includes(err.response.status);
  return !!err.status && ignoreStatuses.includes(err.status);
};

/**
 * expandErrorReasonRequired attempts to expand the error reason for a required
 * field error. It looks for a Mdmlab API error with a `reason` of `"required"`
 * in the `errors` array of the payload. If found, it returns the `name` of the
 * error with the string `"required"` appended. Otherwise, it returns the
 * error reason as is.
 */
export const expandErrorReasonRequired = (err: unknown) => {
  if (isRecordWithDataErrors(err)) {
    const found = err.data.errors.find(
      (e) => isMdmlabApiError(e) && e.reason === "required"
    );
    if (found) {
      return `${(found as IMdmlabApiError).name} required`;
    }
  }
  return getErrorReason(err);
};
