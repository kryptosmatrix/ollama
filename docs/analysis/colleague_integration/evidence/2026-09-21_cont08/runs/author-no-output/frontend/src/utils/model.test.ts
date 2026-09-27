import { describe, expect, it } from "vitest";
import { isCloudModel } from "./model";

describe("isCloudModel", () => {
  it.each(["model:cloud", "model:CLOUD", "model:tag-cloud"])(
    "recognizes an explicit cloud source in %s",
    (model) => expect(isCloudModel(model)).toBe(true),
  );

  it.each(["evilcloud", "my-cloud", "model:local", "model:tag-cloud/path"])(
    "does not misclassify %s as cloud",
    (model) => expect(isCloudModel(model)).toBe(false),
  );
});
