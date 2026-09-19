// Patra SMTP test sender (Hostinger preset by default).
// Config comes from environment / smtp/.env — never hardcode credentials.
//   cp .env.sample .env   (then fill in SMTP_USER / SMTP_PASS)
//   npm install && node index.js
require("dotenv").config();

const nodemailer = require("nodemailer");

function env(name, fallback = "") {
  const v = process.env[name];
  return v === undefined || v === "" ? fallback : v;
}

async function main() {
  const host = env("SMTP_HOST", "smtp.hostinger.com");
  const port = parseInt(env("SMTP_PORT", "465"), 10);
  const secure = env("SMTP_SECURE", port === 465 ? "true" : "false") === "true";
  const user = env("SMTP_USER", "");
  const pass = env("SMTP_PASS", "");
  const from = env("SMTP_FROM", user ? `"Patra" <${user}>` : "");
  const to = env("SMTP_TO", user);
  const subject = env("SMTP_SUBJECT", "Patra SMTP test");

  if (!user || !pass) {
    console.error("Missing SMTP_USER / SMTP_PASS. Copy smtp/.env.sample to smtp/.env and fill them in.");
    process.exitCode = 1;
    return;
  }
  if (!from || !to) {
    console.error("Missing SMTP_FROM / SMTP_TO.");
    process.exitCode = 1;
    return;
  }

  const transporter = nodemailer.createTransport({
    host,
    port,
    secure,
    auth: { user, pass },
  });

  const info = await transporter.sendMail({
    from,
    to,
    subject,
    html: `
      <div style="font-family:Arial, sans-serif; text-align:center; padding:20px;">
        <h2 style="color:#222;">${subject}</h2>
        <p>Sent via Patra (${host}).</p>
      </div>
    `,
    headers: { "X-Mailer": "Patra via Nodemailer" },
  });

  console.log("Email sent successfully:", info.messageId);
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
