import type { ComponentProps } from 'react';
import { NumericFormat, type NumericFormatProps } from 'react-number-format';
import { Input } from './input';

export function PriceInput(props: NumericFormatProps<ComponentProps<typeof Input>>) {
  return (
    <NumericFormat
      type="text"
      inputMode="numeric"
      thousandSeparator=" "
      decimalSeparator="."
      allowedDecimalSeparators={['.', ',']}
      decimalScale={0}
      allowNegative={false}
      {...props}
      customInput={Input}
    />
  );
}
