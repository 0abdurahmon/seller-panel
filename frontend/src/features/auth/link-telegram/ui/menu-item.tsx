import { Send2 } from 'iconsax-reactjs';
import { DropdownMenuItem } from '@/shared/ui';
import { useLinkTelegramDialog } from '../model/dialog.store';

export const LinkTelegramMenuItem = () => {
  const setOpen = useLinkTelegramDialog((state) => state.setOpen);

  return (
    <DropdownMenuItem onClick={() => setOpen(true)}>
      <Send2 />
      Привязать Telegram
    </DropdownMenuItem>
  );
};
