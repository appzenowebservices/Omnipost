const nodemailer = require("nodemailer");

async function main() {
  const transporter = nodemailer.createTransport({
    host: "smtp.hostinger.com",
    port: 465,
    secure: true,
    auth: {
      user: "contact@appzenowebservices.com",
      pass: "%1K#&D)ErQg,aXfi{",
    },
  });

  const info = await transporter.sendMail({
    from: '"Apni Desi Dukaan" <contact@appzenowebservices.com>',
    to: "contact@appzenowebservices.com",
    subject: "It Worked Man!",
    html: `
      <div style="font-family:Arial, sans-serif; text-align:center; padding:20px;">
        <img src="https://apnidesidukaan.com/logo.png" alt="Apni Desi Dukaan" width="120" style="border-radius:10px;" />
        <h2 style="color:#222;">It Worked Man!</h2>
        <p>Heer ne tainu chorr diya.</p>
      </div>
    `,
    headers: { "X-Mailer": "NodeMailer via Hostinger" },
  });

  console.log("✅ Email sent successfully:", info.messageId);
}

main().catch(console.error);
