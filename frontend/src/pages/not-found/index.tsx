import { Link } from 'react-router';
import { Button } from '@/shared/ui';

export const NotFoundPage = () => {
  return (
    <div className="flex flex-col items-start gap-4">
      <p className="text-muted-foreground">Такой страницы нет или она была удалена.</p>
      <Button nativeButton={false} render={<Link to="/" />}>На главную</Button>
    </div>
  );
};
