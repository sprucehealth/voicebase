insert into sku_category (type) values ('visit');
insert into sku (sku_category_id, type) values ((select id from sku_category where type='visit'), 'acne_visit');
