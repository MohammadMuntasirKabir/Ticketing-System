"use client";

import { useState } from "react";
import { Elements, PaymentElement, useElements, useStripe } from "@stripe/react-stripe-js";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getStripe } from "@/lib/stripe";

interface StripePaymentFormProps {
  clientSecret: string;
  amountCents: number;
  currency: string;
  returnUrl: string;
  onSuccess: () => void;
}

function CheckoutForm({ amountCents, currency, returnUrl, onSuccess }: StripePaymentFormProps) {
  const stripe = useStripe();
  const elements = useElements();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handlePay = async () => {
    if (!stripe || !elements) return;
    setBusy(true);
    setError(null);
    const { error: confirmError } = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: returnUrl },
      redirect: "if_required",
    });
    setBusy(false);
    if (confirmError) {
      setError(confirmError.message || "Payment failed");
      return;
    }
    onSuccess();
  };

  return (
    <div className="space-y-4">
      <PaymentElement />
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}
      <Button className="btn-gradient w-full" onClick={handlePay} disabled={busy || !stripe || !elements}>
        {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : `Pay $${(amountCents / 100).toFixed(2)} ${currency}`}
      </Button>
    </div>
  );
}

export function StripePaymentForm(props: StripePaymentFormProps) {
  return (
    <Elements stripe={getStripe()} options={{ clientSecret: props.clientSecret }}>
      <CheckoutForm {...props} />
    </Elements>
  );
}
