/** Thrown when the Graphiti API returns a non-2xx response. */
export class GraphitiApiError extends Error {
  /** HTTP status code. */
  readonly status: number;
  /** Error code from the JSON envelope (mirrors status). */
  readonly code: number;
  /** Optional detail string from the API. */
  readonly detail?: string;

  constructor(status: number, code: number, message: string, detail?: string) {
    super(message);
    this.name = 'GraphitiApiError';
    this.status = status;
    this.code = code;
    this.detail = detail;
  }
}

/** Thrown when a network error prevents the request from completing. */
export class GraphitiNetworkError extends Error {
  readonly cause?: Error;

  constructor(message: string, cause?: Error) {
    super(message);
    this.name = 'GraphitiNetworkError';
    this.cause = cause;
  }
}
