export async function allPages<T>(getPage: (page: number) => Promise<T[]>): Promise<T[]> {
  const items: T[] = [];
  for (let page = 1; ; page += 1) {
    const pageItems = await getPage(page);
    if (pageItems.length === 0) return items;
    items.push(...pageItems);
  }
}
