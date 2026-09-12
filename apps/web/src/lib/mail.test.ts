// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import type { CalApi } from "@cal/api-client";
import { fileToBase64, httpMail, mailBackend, nativeMail } from "./mail";

function stubApi(over: Partial<Record<keyof CalApi, unknown>> = {}): CalApi {
  return {
    mailAccounts: async () => [{ id: "a1", email: "me@example.com" }],
    upload: async (f: File) => ({ url: "", name: `stored-${f.name}`, markdown: "" }),
    mailSend: async () => {},
    ...over,
  } as unknown as CalApi;
}

describe("mailBackend", () => {
  it("returns the http transport off the Android shell", async () => {
    const backend = mailBackend(stubApi());
    const accounts = await backend.accounts();
    expect(accounts[0].email).toBe("me@example.com");
  });

  it("addAccount forwards custom ports and the insecure flag", async () => {
    let seen: Record<string, unknown> = {};
    const api = stubApi({
      createMailAccount: async (input: Record<string, unknown>) => {
        seen = input;
        return { id: "a2" };
      },
    });
    await httpMail(api).addAccount({
      email: "me@example.com",
      imapHost: "imap.home.lan", imapPort: 143,
      smtpHost: "smtp.home.lan", smtpPort: 2525,
      password: "pw", insecure: true,
    });
    expect(seen).toMatchObject({ imapPort: 143, smtpPort: 2525, insecure: true });
  });

  it("httpMail.send uploads files then sends stored names", async () => {
    let sent: { attachments?: string[] } = {};
    const api = stubApi({ mailSend: async (_id: string, input: { attachments?: string[] }) => { sent = input; } });
    await httpMail(api).send("a1", {
      to: "you@example.com",
      subject: "hi",
      text: "body",
      files: [new File(["x"], "note.txt", { type: "text/plain" })],
    });
    expect(sent.attachments).toEqual(["stored-note.txt"]);
  });
});

describe("fileToBase64", () => {
  it("encodes file bytes for the native bridge", async () => {
    const out = await fileToBase64(new File(["hello"], "a.txt"));
    expect(out).toBe("aGVsbG8=");
  });
});

describe("nativeMail", () => {
  it("rejects off-device rather than pretending", async () => {
    await expect(nativeMail.accounts()).rejects.toThrow();
  });
});
