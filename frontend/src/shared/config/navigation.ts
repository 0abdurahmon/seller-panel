import { Box, Category, ReceiptText, Shop, Wallet, type Icon } from 'iconsax-reactjs';

export interface MenuItem {
  label: string;
  href: string;
  icon: Icon;
}

export const mainMenuItems: MenuItem[] = [
  { label: 'Дашборд', href: '/', icon: Category },
  { label: 'Товары', href: '/products', icon: Box },
  { label: 'Заказы', href: '/orders', icon: ReceiptText },
  { label: 'Баланс', href: '/balance', icon: Wallet },
  { label: 'Магазин', href: '/shop', icon: Shop },
];

export const bottomMenuItems: MenuItem[] = [];
