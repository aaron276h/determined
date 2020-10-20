from gan import GAN, create_generator, create_discriminator, get_dataset
import tensorflow as tf
import numpy as np
import h5py
from tensorflow.python.keras.saving.hdf5_format import save_optimizer_weights_to_hdf5_group, load_optimizer_weights_from_hdf5_group

def train():
    model = GAN(
        discriminator=create_discriminator(),
        generator=create_generator(128),
        latent_dim=128
    )

    d_optimizer = tf.keras.optimizers.Adam(learning_rate=0.0003)
    g_optimizer = tf.keras.optimizers.Adam(learning_rate=0.0003)

    model.compile(
        d_optimizer=d_optimizer,
        g_optimizer=g_optimizer,
        loss_fn=tf.keras.losses.BinaryCrossentropy(from_logits=True),
    )

    ds = get_dataset()
    ds = ds.shuffle(1024).batch(64)

    d_optimizer._create_all_weights(model.trainable_variables)
    model.fit(
        x=ds.take(5),
        epochs=1
    )

    """
    save_format = "h5"
    if save_format == "h5":
        model.save("my-model.h5", save_format="h5")
    else:
        model.save("my-model", save_format="tf")
    """

    model.save_weights("my-model-weights.h5", save_format="h5")
    print("optimizers weights: ", d_optimizer.get_weights(), len(d_optimizer.get_weights()))

    with h5py.File("my-optimizer-weights.h5", "w") as h5file:
        opt_group = h5file.create_group("opt")
        save_optimizer_weights_to_hdf5_group(opt_group, d_optimizer)
        #print(opt_group["optimizer_weights"]["Adam"].keys())
    #print("Optimizer weights: ", d_optimizer.get_weights())


def load():
    model = GAN(
        discriminator=create_discriminator(),
        generator=create_generator(128),
        latent_dim=128
    )

    d_optimizer = tf.keras.optimizers.Adam(learning_rate=0.0003)
    g_optimizer = tf.keras.optimizers.Adam(learning_rate=0.0003)
    d_optimizer._create_all_weights(model.trainable_variables)
    g_optimizer._create_all_weights(model.trainable_variables)

    model.compile(
        d_optimizer=d_optimizer,
        g_optimizer=g_optimizer,
        loss_fn=tf.keras.losses.BinaryCrossentropy(from_logits=True),
    )

    ds = get_dataset()
    ds = ds.shuffle(1024).batch(64)

    model.load_weights("my-model-weights.h5")
    #print("My model: ", model.get_weights())
    #print("optimizer: ", d_optimizer.get_weights())

    with h5py.File("my-optimizer-weights.h5", 'r') as h5f:
        weights_group = h5f["opt"]
        optimizer_values = load_optimizer_weights_from_hdf5_group(weights_group)
        d_optimizer.set_weights(optimizer_values)
    print("optimizer info: ", d_optimizer.get_weights(), len(d_optimizer.get_weights()))

if __name__ == "__main__":
    #train()
    load()
