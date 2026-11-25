import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Copy, Export } from 'iconsax-reactjs';
import { getApiErrorMessage, getAuth } from '@/shared/api';
import { formatDateTime } from '@/shared/lib';
import {
  Alert,
  AlertDescription,
  Button,
  CheckIcon,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Spinner,
} from '@/shared/ui';
import { useLinkTelegramDialog } from '../model/dialog.store';

const api = getAuth();

export const LinkTelegramDialog = () => {
  const open = useLinkTelegramDialog((state) => state.open);
  const setOpen = useLinkTelegramDialog((state) => state.setOpen);
  const [copied, setCopied] = useState(false);

  const link = useMutation({
    mutationFn: () => api.postAuthTelegramLink(),
  });

  useEffect(() => {
    if (open) {
      link.mutate();
    } else {
      link.reset();
      setCopied(false);
    }
    // Only trigger when dialog opens or closes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const handleCopy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for environments where clipboard API is restricted
      const textarea = document.createElement('textarea');
      textarea.value = url;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Привязка Telegram</DialogTitle>
          <DialogDescription>
            Откройте ссылку в Telegram и нажмите Start, чтобы получать коды для входа. Повторная
            привязка заменит предыдущий чат.
          </DialogDescription>
        </DialogHeader>

        {link.isPending && (
          <div className="flex items-center gap-2.5 py-2 text-sm text-muted-foreground">
            <Spinner />
            <span>Создаём одноразовую ссылку…</span>
          </div>
        )}

        {link.isError && (
          <Alert variant="destructive">
            <AlertDescription>{getApiErrorMessage(link.error)}</AlertDescription>
          </Alert>
        )}

        {link.isSuccess && (
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <Input
                readOnly
                value={link.data.url}
                className="select-all font-mono text-xs"
                onFocus={(e) => e.target.select()}
                aria-label="Ссылка для привязки Telegram"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={() => handleCopy(link.data.url)}
                aria-label={copied ? 'Ссылка скопирована' : 'Скопировать ссылку'}
                title={copied ? 'Скопировано!' : 'Скопировать ссылку'}
              >
                {copied ? <CheckIcon className="text-success" /> : <Copy />}
              </Button>
              <Button
                variant="default"
                size="icon"
                nativeButton={false}
                aria-label="Открыть в Telegram"
                title="Открыть в Telegram"
                render={
                  <a href={link.data.url} target="_blank" rel="noopener noreferrer" />
                }
              >
                <Export />
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Ссылка одноразовая и действует до {formatDateTime(link.data.expires_at)}.
            </p>
          </div>
        )}

        {link.isError && (
          <DialogFooter>
            <Button type="button" variant="secondary" onClick={() => link.mutate()}>
              Попробовать снова
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
};
