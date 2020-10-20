from determined.keras import TFKerasTrial, InputData
import tensorflow as tf

from gan import GAN, get_dataset, create_discriminator, create_generator


class GanTrial(TFKerasTrial):
    def __init__(self, context):
        self.context = context

    def build_model(self) -> tf.keras.models.Model:
        model = GAN(
            discriminator=create_discriminator(),
            generator=create_generator(self.context.get_hparam("latent_dim")),
            latent_dim=self.context.get_hparam("latent_dim"),
        )
        model = self.context.wrap_model(model)

        d_optimizer = self.context.wrap_optimizer(tf.keras.optimizers.Adam(learning_rate=0.0003))
        g_optimizer = self.context.wrap_optimizer(tf.keras.optimizers.Adam(learning_rate=0.0003))

        model.compile(
            d_optimizer=d_optimizer,
            g_optimizer=g_optimizer,
            loss_fn=tf.keras.losses.BinaryCrossentropy(from_logits=True),
        )

        return model

    def build_training_data_loader(self) -> InputData:
        ds = get_dataset()
        ds = self.context.wrap_dataset(ds)
        ds = ds.shuffle(buffer_size=1024).batch(self.context.get_per_slot_batch_size())
        return ds

    def build_validation_data_loader(self) -> InputData:
        ds = get_dataset()
        ds = self.context.wrap_dataset(ds)
        ds = ds.shuffle(buffer_size=1024).batch(self.context.get_per_slot_batch_size())
        return ds
