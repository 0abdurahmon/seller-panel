import { Skeleton } from 'skullet/react';
import { useShop, type Shop } from '@/entities/shop';
import { EditShopForm } from '@/features/shop/edit-shop';
import { getApiErrorMessage } from '@/shared/api';
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
} from '@/shared/ui';

const SAMPLE_SHOP: Shop = {
  id: -1,
  name: 'Dona Electronics',
  description: 'Электроника, аксессуары и умные устройства с доставкой по городу.',
  contact_phone: '+998901234567',
  pickup_address: 'г. Ташкент, ул. Амира Темура, 1',
  logo_url: null,
};

export const ShopPage = () => {
  const shop = useShop();

  if (shop.isError && !shop.data) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Не удалось загрузить профиль магазина</AlertTitle>
        <AlertDescription>{getApiErrorMessage(shop.error)}</AlertDescription>
        <AlertAction>
          <Button size="sm" variant="outline" onClick={() => shop.refetch()}>
            Повторить
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  return (
    <Skeleton
      id="shop-profile"
      label="Загрузка профиля магазина"
      data={shop.isLoading ? null : shop.data}
      sample={SAMPLE_SHOP}
    >
      {(data) => <EditShopForm key={data.id} shop={data} />}
    </Skeleton>
  );
};
